package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/helloodokai/charter/internal/config"
)

type OllamaClient struct {
	host   string
	apiKey string
	client *http.Client
	local  bool
}

func NewOllamaCloudClient(cfg config.OllamaConfig) *OllamaClient {
	return &OllamaClient{
		host:   cfg.Host,
		apiKey: cfg.APIKey,
		local:  false,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func NewOllamaLocalClient(cfg config.OllamaConfig) *OllamaClient {
	return &OllamaClient{
		host:   cfg.Host,
		apiKey: "",
		local:  true,
		client: &http.Client{Timeout: 300 * time.Second},
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
	Model              string `json:"model"`
	Done               bool   `json:"done"`
	PromptEvalCount    int    `json:"prompt_eval_count"`
	EvalCount          int    `json:"eval_count"`
	TotalDuration      int64  `json:"total_duration"`
}

func (c *OllamaClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	ollamaReq := ollamaRequest{
		Model:  req.Model,
		Stream: false,
		System: req.System,
		Options: ollamaOptions{
			NumPredict: req.MaxTokens,
		},
	}
	for _, m := range req.Messages {
		ollamaReq.Messages = append(ollamaReq.Messages, ollamaMsg{
			Role:    m.Role,
			Content: m.Content,
		})
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

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama %s: %s: %s", endpoint, resp.Status, string(respBody))
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

func (c *OllamaClient) Name() string {
	if c.local {
		return "ollama-local"
	}
	return "ollama-cloud"
}