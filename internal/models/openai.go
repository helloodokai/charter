package models

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/helloodokai/charter/internal/config"
)

type OpenAIClient struct {
	client *openai.Client
	apiKey string
}

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

func (c *OpenAIClient) Name() string {
	return "openai"
}