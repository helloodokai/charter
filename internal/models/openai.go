package models

import (
	"context"
	"fmt"
	"io"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/helloodokai/charter/internal/config"
)

// OpenAIClient implements Client for the OpenAI API.
type OpenAIClient struct {
	client *openai.Client
	apiKey string
}

// NewOpenAIClient creates a new OpenAI client from the given config.
func NewOpenAIClient(cfg config.OpenAIConfig) *OpenAIClient {
	opts := []option.RequestOption{}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	client := openai.NewClient(opts...)
	return &OpenAIClient{
		client: &client,
		apiKey: cfg.APIKey,
	}
}

// Complete sends a non-streaming completion request to OpenAI.
func (c *OpenAIClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var messages []openai.ChatCompletionMessageParamUnion
	if req.System != "" {
		messages = append(messages, openai.SystemMessage(req.System))
	}
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, openai.UserMessage(m.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(m.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    req.Model,
		Messages: messages,
	}

	if req.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(req.MaxTokens))
	}

	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}

	if len(completion.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices returned")
	}

	return &CompletionResponse{
		Content: completion.Choices[0].Message.Content,
		Model:   string(completion.Model),
		Usage: Usage{
			InputTokens:  int(completion.Usage.PromptTokens),
			OutputTokens: int(completion.Usage.CompletionTokens),
		},
	}, nil
}

// Stream sends a streaming completion request to OpenAI, writing tokens to w.
func (c *OpenAIClient) Stream(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error) {
	var messages []openai.ChatCompletionMessageParamUnion
	if req.System != "" {
		messages = append(messages, openai.SystemMessage(req.System))
	}
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, openai.UserMessage(m.Content))
		case "assistant":
			messages = append(messages, openai.AssistantMessage(m.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:    req.Model,
		Messages: messages,
	}

	if req.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(req.MaxTokens))
	}

	stream := c.client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)

		if len(chunk.Choices) > 0 {
			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				fmt.Fprint(w, delta.Content)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("openai stream: %w", err)
	}

	result := acc.ChatCompletion
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openai stream: no choices returned")
	}

	return &CompletionResponse{
		Content: result.Choices[0].Message.Content,
		Model:   string(result.Model),
		Usage: Usage{
			InputTokens:  int(result.Usage.PromptTokens),
			OutputTokens: int(result.Usage.CompletionTokens),
		},
	}, nil
}

// Name returns the provider name for the OpenAI client.
func (c *OpenAIClient) Name() string {
	return "openai"
}