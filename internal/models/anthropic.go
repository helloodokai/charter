package models

import (
	"context"
	"fmt"
	"io"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/helloodokai/charter/internal/config"
)

type AnthropicClient struct {
	client *anthropic.Client
	apiKey string
}

func NewAnthropicClient(cfg config.AnthropicConfig) *AnthropicClient {
	opts := []option.RequestOption{}
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}
	client := anthropic.NewClient(opts...)
	return &AnthropicClient{
		client: &client,
		apiKey: cfg.APIKey,
	}
}

func (c *AnthropicClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var messages []anthropic.MessageParam
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(m.Content),
			))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(m.Content),
			))
		}
	}

	maxTokens := int64(4096)
	if req.MaxTokens > 0 {
		maxTokens = int64(req.MaxTokens)
	}

	params := anthropic.MessageNewParams{
		MaxTokens: maxTokens,
		Model:     req.Model,
		Messages:  messages,
	}

	if req.System != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.System},
		}
	}

	msg, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic request: %w", err)
	}

	content := ""
	for _, block := range msg.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &CompletionResponse{
		Content: content,
		Model:   string(msg.Model),
		Usage: Usage{
			InputTokens:  int(msg.Usage.InputTokens),
			OutputTokens: int(msg.Usage.OutputTokens),
		},
	}, nil
}

func (c *AnthropicClient) Stream(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error) {
	var messages []anthropic.MessageParam
	for _, m := range req.Messages {
		switch m.Role {
		case "user":
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(m.Content),
			))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(m.Content),
			))
		}
	}

	maxTokens := int64(4096)
	if req.MaxTokens > 0 {
		maxTokens = int64(req.MaxTokens)
	}

	params := anthropic.MessageNewParams{
		MaxTokens: maxTokens,
		Model:     req.Model,
		Messages:  messages,
	}

	if req.System != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.System},
		}
	}

	stream := c.client.Messages.NewStreaming(ctx, params)

	var fullContent string
	var model string
	var inputTokens, outputTokens int

	for stream.Next() {
		event := stream.Current()

		switch event.Type {
		case "message_start":
			msg := event.AsMessageStart()
			model = string(msg.Message.Model)
			inputTokens = int(msg.Message.Usage.InputTokens)

		case "content_block_delta":
			delta := event.AsContentBlockDelta()
			textDelta := delta.Delta.AsTextDelta()
			if textDelta.Text != "" {
				fullContent += textDelta.Text
				fmt.Fprint(w, textDelta.Text)
			}

		case "message_delta":
			msgDelta := event.AsMessageDelta()
			outputTokens = int(msgDelta.Usage.OutputTokens)
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("anthropic stream: %w", err)
	}

	return &CompletionResponse{
		Content: fullContent,
		Model:   model,
		Usage: Usage{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
		},
	}, nil
}

func (c *AnthropicClient) Name() string {
	return "anthropic"
}