package models

import (
	"context"
)

type Tier string

const (
	Cheap    Tier = "cheap"
	Mid      Tier = "mid"
	Frontier Tier = "frontier"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	System   string    `json:"system,omitempty"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

type CompletionResponse struct {
	Content string
	Model   string
	Usage   Usage
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}

type Client interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Name() string
}

type Provider string

const (
	ProviderOllamaCloud Provider = "ollama_cloud"
	ProviderOllamaLocal Provider = "ollama_local"
	ProviderAnthropic   Provider = "anthropic"
	ProviderOpenAI      Provider = "openai"
)

