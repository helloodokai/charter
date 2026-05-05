package models

import (
	"context"
	"io"
)

// Tier represents a model quality tier for routing LLM calls.
type Tier string

// Cheap is the lowest-cost, least-capable model tier.
const (
	Cheap    Tier = "cheap"
	Mid      Tier = "mid"
	Frontier Tier = "frontier"
)

// Message represents a single message in a completion conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest represents a request to generate a completion from an LLM.
type CompletionRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	System    string    `json:"system,omitempty"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

// CompletionResponse represents the result of an LLM completion call.
type CompletionResponse struct {
	Content string
	Model   string
	Usage   Usage
}

// Usage tracks token consumption for a completion request.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// StreamEvent represents a single event in a streaming completion response.
type StreamEvent struct {
	Type    StreamEventType
	Content string
	Usage   *Usage
}

// StreamEventType identifies the type of a streaming event.
type StreamEventType int

// StreamToken indicates a token chunk in the stream.
const (
	StreamToken StreamEventType = iota
	StreamDone
)

// StreamingCallback is a function invoked for each event during streaming.
type StreamingCallback func(event StreamEvent)

// Client defines the interface for LLM completion providers.
type Client interface {
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	Stream(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error)
	Name() string
}

// Provider identifies an LLM backend provider.
type Provider string

// ProviderOllamaCloud is the Ollama cloud-hosted provider.
const (
	ProviderOllamaCloud Provider = "ollama_cloud"
	ProviderOllamaLocal Provider = "ollama_local"
	ProviderAnthropic   Provider = "anthropic"
	ProviderOpenAI      Provider = "openai"
)