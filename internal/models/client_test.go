package models

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockClient struct {
	CompleteFn func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	StreamFn   func(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error)
	name       string
}

func (m *MockClient) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if m.CompleteFn != nil {
		return m.CompleteFn(ctx, req)
	}
	return &CompletionResponse{Content: "mock complete", Model: req.Model}, nil
}

func (m *MockClient) Stream(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error) {
	if m.StreamFn != nil {
		return m.StreamFn(ctx, req, w)
	}
	return &CompletionResponse{Content: "mock stream", Model: req.Model}, nil
}

func (m *MockClient) Name() string {
	return m.name
}

var _ Client = (*MockClient)(nil)

func TestMockClientImplementsClient(t *testing.T) {
	var c Client = &MockClient{name: "mock"}
	require.Implements(t, (*Client)(nil), c)
}

func TestMockClientCompleteDefault(t *testing.T) {
	c := &MockClient{name: "mock"}
	resp, err := c.Complete(context.Background(), CompletionRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	require.NoError(t, err)
	require.Equal(t, "mock complete", resp.Content)
	require.Equal(t, "test-model", resp.Model)
}

func TestMockClientStreamDefault(t *testing.T) {
	c := &MockClient{name: "mock"}
	buf := &bytes.Buffer{}
	resp, err := c.Stream(context.Background(), CompletionRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, buf)
	require.NoError(t, err)
	require.Equal(t, "mock stream", resp.Content)
	require.Equal(t, "test-model", resp.Model)
}

func TestMockClientStreamWritesTokens(t *testing.T) {
	tokens := []string{"Hello", " ", "world", "!"}
	c := &MockClient{
		name: "mock",
		StreamFn: func(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error) {
			for _, tok := range tokens {
				_, _ = w.Write([]byte(tok))
			}
			return &CompletionResponse{
				Content: "Hello world!",
				Model:   req.Model,
				Usage:   Usage{InputTokens: 10, OutputTokens: 4},
			}, nil
		},
	}

	buf := &bytes.Buffer{}
	resp, err := c.Stream(context.Background(), CompletionRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, buf)

	require.NoError(t, err)
	require.Equal(t, "Hello world!", buf.String())
	require.Equal(t, "Hello world!", resp.Content)
	require.Equal(t, 10, resp.Usage.InputTokens)
	require.Equal(t, 4, resp.Usage.OutputTokens)
}

func TestMockClientStreamError(t *testing.T) {
	c := &MockClient{
		name: "mock-err",
		StreamFn: func(ctx context.Context, req CompletionRequest, w io.Writer) (*CompletionResponse, error) {
			_, _ = w.Write([]byte("partial"))
			return nil, io.ErrUnexpectedEOF
		},
	}

	buf := &bytes.Buffer{}
	resp, err := c.Stream(context.Background(), CompletionRequest{
		Model:    "test-model",
		Messages: []Message{{Role: "user", Content: "hello"}},
	}, buf)

	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, "partial", buf.String())
}

func TestMockClientCompleteCustom(t *testing.T) {
	c := &MockClient{
		name: "mock",
		CompleteFn: func(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
			return &CompletionResponse{
				Content: "custom response",
				Model:   req.Model,
				Usage:   Usage{InputTokens: 5, OutputTokens: 10},
			}, nil
		},
	}

	resp, err := c.Complete(context.Background(), CompletionRequest{
		Model: "custom-model",
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "custom response", resp.Content)
	require.Equal(t, "custom-model", resp.Model)
	require.Equal(t, 5, resp.Usage.InputTokens)
	require.Equal(t, 10, resp.Usage.OutputTokens)
}

func TestMockClientName(t *testing.T) {
	c := &MockClient{name: "test-provider"}
	require.Equal(t, "test-provider", c.Name())
}

func TestClientInterfaceRequiresComplete(t *testing.T) {
	c := &MockClient{name: "mock"}
	resp, err := c.Complete(context.Background(), CompletionRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "hi"}},
		System:   "sys",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestStreamEventTypes(t *testing.T) {
	require.Equal(t, StreamToken, StreamEventType(0))
	require.Equal(t, StreamDone, StreamEventType(1))
}

func TestTierConstants(t *testing.T) {
	require.Equal(t, Tier("cheap"), Cheap)
	require.Equal(t, Tier("mid"), Mid)
	require.Equal(t, Tier("frontier"), Frontier)
}

func TestProviderConstants(t *testing.T) {
	require.Equal(t, Provider("ollama_cloud"), ProviderOllamaCloud)
	require.Equal(t, Provider("ollama_local"), ProviderOllamaLocal)
	require.Equal(t, Provider("anthropic"), ProviderAnthropic)
	require.Equal(t, Provider("openai"), ProviderOpenAI)
}

func TestCompletionRequestFields(t *testing.T) {
	req := CompletionRequest{
		Model:     "gpt-4",
		Messages:  []Message{{Role: "user", Content: "test"}},
		System:    "system prompt",
		MaxTokens: 100,
	}
	require.Equal(t, "gpt-4", req.Model)
	require.Len(t, req.Messages, 1)
	require.Equal(t, "system prompt", req.System)
	require.Equal(t, 100, req.MaxTokens)
}

func TestCompletionResponseFields(t *testing.T) {
	resp := &CompletionResponse{
		Content: "response text",
		Model:   "gpt-4",
		Usage:   Usage{InputTokens: 10, OutputTokens: 20},
	}
	require.Equal(t, "response text", resp.Content)
	require.Equal(t, "gpt-4", resp.Model)
	require.Equal(t, 10, resp.Usage.InputTokens)
	require.Equal(t, 20, resp.Usage.OutputTokens)
}