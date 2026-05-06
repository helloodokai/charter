package models

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/helloodokai/charter/internal/config"
)

// OllamaClient implements Client for the Ollama API.
type OllamaClient struct {
	host   string
	apiKey string
	client *http.Client
	local  bool
}

// NewOllamaCloudClient creates a new Ollama client targeting the cloud-hosted instance.
func NewOllamaCloudClient(cfg config.OllamaConfig) *OllamaClient {
	return &OllamaClient{
		host:   cfg.Host,
		apiKey: cfg.APIKey,
		local:  false,
		client: &http.Client{Timeout: 300 * time.Second},
	}
}

// NewOllamaLocalClient creates a new Ollama client targeting a local instance.
func NewOllamaLocalClient(cfg config.OllamaConfig) *OllamaClient {
	return &OllamaClient{
		host:   cfg.Host,
		apiKey: "",
		local:  true,
		client: &http.Client{Timeout: 600 * time.Second},
	}
}

type ollamaRequest struct {
	Model    string        `json:"model"`
	Messages []ollamaMsg   `json:"messages"`
	Stream   bool          `json:"stream"`
	System   string        `json:"system,omitempty"`
	Options  ollamaOptions `json:"options,omitempty"`
}

type ollamaOptions struct {
	NumPredict int `json:"num_predict,omitempty"`
}

type ollamaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message struct {
		Content string `json:"content"`
		Role    string `json:"role"`
	} `json:"message"`
	Model           string `json:"model"`
	Done            bool   `json:"done"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
	TotalDuration   int64  `json:"total_duration"`
}

func (c *OllamaClient) doRequest(ctx context.Context, req CompletionRequest, stream bool) (*http.Response, error) {
	system := req.System
	if req.WebSearch {
		lastMsg := ""
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				lastMsg = req.Messages[i].Content
				break
			}
		}
		if lastMsg != "" {
			results, err := WebSearch(ctx, lastMsg)
			if err == nil && results != "" {
				system += "\n\nWeb search results:\n" + results
			}
		}
	}

	ollamaReq := ollamaRequest{
		Model:  req.Model,
		Stream: stream,
		System: system,
		Options: ollamaOptions{
			NumPredict: req.MaxTokens,
		},
	}
	for _, m := range req.Messages {
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMsg(m))
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshalling ollama request: %w", err)
	}

	endpoint := c.host + "/api/chat"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	return c.client.Do(httpReq)
}

// Complete sends a non-streaming completion request to Ollama.
func (c *OllamaClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	resp, err := c.doRequest(ctx, req, false)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama %s: %s", resp.Status, string(respBody))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decoding ollama response: %w", err)
	}

	return &CompletionResponse{
		Content: ollamaResp.Message.Content,
		Model:   ollamaResp.Model,
		Usage: Usage{
			InputTokens:  ollamaResp.PromptEvalCount,
			OutputTokens: ollamaResp.EvalCount,
		},
	}, nil
}

// Stream sends a streaming completion request to Ollama, writing tokens to w.
func (c *OllamaClient) Stream(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error) {
	resp, err := c.doRequest(ctx, req, true)
	if err != nil {
		return nil, fmt.Errorf("ollama stream request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama stream %s: %s", resp.Status, string(respBody))
	}

	var fullContent strings.Builder
	var model string
	var promptEval, evalCount int

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var chunk ollamaResponse
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}

		if chunk.Model != "" {
			model = chunk.Model
		}

		token := chunk.Message.Content
		if token != "" {
			fullContent.WriteString(token)
			fmt.Fprint(w, token)
		}

		if chunk.Done {
			promptEval = chunk.PromptEvalCount
			evalCount = chunk.EvalCount
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ollama stream: %w", err)
	}

	return &CompletionResponse{
		Content: fullContent.String(),
		Model:   model,
		Usage: Usage{
			InputTokens:  promptEval,
			OutputTokens: evalCount,
		},
	}, nil
}

// Name returns the provider name for the Ollama client.
func (c *OllamaClient) Name() string {
	if c.local {
		return "ollama-local"
	}
	return "ollama-cloud"
}