package routing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/models"
)

type mockClient struct {
	completeFn func(ctx context.Context, req models.CompletionRequest) (*models.CompletionResponse, error)
	streamFn  func(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error)
	name      string
}

func (m *mockClient) Complete(ctx context.Context, req models.CompletionRequest) (*models.CompletionResponse, error) {
	if m.completeFn != nil {
		return m.completeFn(ctx, req)
	}
	return &models.CompletionResponse{Content: "complete", Model: req.Model}, nil
}

func (m *mockClient) Stream(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
	if m.streamFn != nil {
		return m.streamFn(ctx, req, w)
	}
	return &models.CompletionResponse{Content: "stream", Model: req.Model}, nil
}

func (m *mockClient) Name() string {
	return m.name
}

func newTestRouter(clients map[models.Provider]models.Client, profile config.ProfileConfig) *Router {
	var fallback *config.ProfileConfig
	return &Router{
		clients:  clients,
		profile:  profile,
		fallback: fallback,
	}
}

func TestStreamDelegatesToClient(t *testing.T) {
	mock := &mockClient{
		name: "test-mock",
		streamFn: func(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
			fmt.Fprint(w, "hello world")
			return &models.CompletionResponse{
				Content: "hello world",
				Model:   req.Model,
				Usage:   models.Usage{InputTokens: 5, OutputTokens: 2},
			}, nil
		},
	}

	profile := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "test", Name: "test-cheap"},
		Mid:      config.ModelRef{Provider: "test", Name: "test-mid"},
		Frontier: config.ModelRef{Provider: "test", Name: "test-frontier"},
	}

	clients := map[models.Provider]models.Client{
		"test": mock,
	}

	r := newTestRouter(clients, profile)
	buf := &bytes.Buffer{}

	resp, err := r.Stream(context.Background(), models.Mid, models.CompletionRequest{
		Messages: []models.Message{{Role: "user", Content: "hi"}},
	}, buf)

	require.NoError(t, err)
	require.Equal(t, "hello world", buf.String())
	require.Equal(t, "hello world", resp.Content)
	require.Equal(t, "test-mid", resp.Model)
	require.Equal(t, 5, resp.Usage.InputTokens)
	require.Equal(t, 2, resp.Usage.OutputTokens)
}

func TestStreamUsesCorrectTier(t *testing.T) {
	calls := []string{}
	clients := map[models.Provider]models.Client{}

	tiers := []struct {
		provider models.Provider
		ref      config.ModelRef
	}{
		{"cheap-prov", config.ModelRef{Provider: "cheap-prov", Name: "cheap-model"}},
		{"mid-prov", config.ModelRef{Provider: "mid-prov", Name: "mid-model"}},
		{"frontier-prov", config.ModelRef{Provider: "frontier-prov", Name: "frontier-model"}},
	}
	for _, tier := range tiers {
		prov := tier.provider
		modelName := tier.ref.Name
		clients[tier.provider] = &mockClient{
			name: string(prov),
			streamFn: func(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
				calls = append(calls, modelName)
				return &models.CompletionResponse{Content: "ok", Model: req.Model}, nil
			},
		}
	}

	profile := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "cheap-prov", Name: "cheap-model"},
		Mid:      config.ModelRef{Provider: "mid-prov", Name: "mid-model"},
		Frontier: config.ModelRef{Provider: "frontier-prov", Name: "frontier-model"},
	}

	r := newTestRouter(clients, profile)
	buf := &bytes.Buffer{}

	_, err := r.Stream(context.Background(), models.Cheap, models.CompletionRequest{}, buf)
	require.NoError(t, err)

	_, err = r.Stream(context.Background(), models.Mid, models.CompletionRequest{}, buf)
	require.NoError(t, err)

	_, err = r.Stream(context.Background(), models.Frontier, models.CompletionRequest{}, buf)
	require.NoError(t, err)

	require.Equal(t, []string{"cheap-model", "mid-model", "frontier-model"}, calls)
}

func TestStreamFallbackOnPrimaryError(t *testing.T) {
	primaryErr := fmt.Errorf("primary failed")
	primaryCalled := false
	fallbackCalled := false

	clients := map[models.Provider]models.Client{
		"test": &mockClient{
			name: "test",
			streamFn: func(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
				primaryCalled = true
				return nil, primaryErr
			},
		},
		"fallback": &mockClient{
			name: "fallback",
			streamFn: func(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
				fallbackCalled = true
				fmt.Fprint(w, "fallback response")
				return &models.CompletionResponse{Content: "fallback response", Model: req.Model}, nil
			},
		},
	}

	profile := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "test", Name: "test-model"},
		Mid:      config.ModelRef{Provider: "test", Name: "test-model"},
		Frontier: config.ModelRef{Provider: "test", Name: "test-model"},
	}

	fallback := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "fallback", Name: "fallback-cheap"},
		Mid:      config.ModelRef{Provider: "fallback", Name: "fallback-mid"},
		Frontier: config.ModelRef{Provider: "fallback", Name: "fallback-frontier"},
	}

	r := &Router{
		clients:  clients,
		profile:  profile,
		fallback: &fallback,
	}

	buf := &bytes.Buffer{}
	resp, err := r.Stream(context.Background(), models.Mid, models.CompletionRequest{}, buf)

	require.NoError(t, err)
	require.True(t, primaryCalled)
	require.True(t, fallbackCalled)
	require.Equal(t, "fallback response", resp.Content)
	require.Equal(t, "fallback-mid", resp.Model)
}

func TestStreamErrorNoFallback(t *testing.T) {
	clients := map[models.Provider]models.Client{
		"test": &mockClient{
			name: "test",
			streamFn: func(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
				return nil, fmt.Errorf("stream error")
			},
		},
	}

	profile := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "test", Name: "test-model"},
		Mid:      config.ModelRef{Provider: "test", Name: "test-model"},
		Frontier: config.ModelRef{Provider: "test", Name: "test-model"},
	}

	r := newTestRouter(clients, profile)
	buf := &bytes.Buffer{}

	resp, err := r.Stream(context.Background(), models.Mid, models.CompletionRequest{}, buf)
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestStreamNoClientForProvider(t *testing.T) {
	clients := map[models.Provider]models.Client{}

	profile := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "missing", Name: "missing-cheap"},
		Mid:      config.ModelRef{Provider: "missing", Name: "missing-mid"},
		Frontier: config.ModelRef{Provider: "missing", Name: "missing-frontier"},
	}

	r := newTestRouter(clients, profile)
	buf := &bytes.Buffer{}

	resp, err := r.Stream(context.Background(), models.Mid, models.CompletionRequest{}, buf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no client for provider")
	require.Nil(t, resp)
}

func TestStreamSetsModelOnRequest(t *testing.T) {
	receivedModel := ""
	clients := map[models.Provider]models.Client{
		"test": &mockClient{
			name: "test",
			streamFn: func(ctx context.Context, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
				receivedModel = req.Model
				return &models.CompletionResponse{Content: "ok", Model: req.Model}, nil
			},
		},
	}

	profile := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "test", Name: "specific-cheap-model"},
		Mid:      config.ModelRef{Provider: "test", Name: "specific-mid-model"},
		Frontier: config.ModelRef{Provider: "test", Name: "specific-frontier-model"},
	}

	r := newTestRouter(clients, profile)
	buf := &bytes.Buffer{}

	_, err := r.Stream(context.Background(), models.Frontier, models.CompletionRequest{}, buf)
	require.NoError(t, err)
	require.Equal(t, "specific-frontier-model", receivedModel)
}

func TestCompleteDelegatesToClient(t *testing.T) {
	clients := map[models.Provider]models.Client{
		"test": &mockClient{
			name: "test",
			completeFn: func(ctx context.Context, req models.CompletionRequest) (*models.CompletionResponse, error) {
				return &models.CompletionResponse{
					Content: "complete response",
					Model:   req.Model,
					Usage:   models.Usage{InputTokens: 3, OutputTokens: 5},
				}, nil
			},
		},
	}

	profile := config.ProfileConfig{
		Cheap:    config.ModelRef{Provider: "test", Name: "test-cheap"},
		Mid:      config.ModelRef{Provider: "test", Name: "test-mid"},
		Frontier: config.ModelRef{Provider: "test", Name: "test-frontier"},
	}

	r := newTestRouter(clients, profile)

	resp, err := r.Complete(context.Background(), models.Cheap, models.CompletionRequest{
		Messages: []models.Message{{Role: "user", Content: "hi"}},
	})

	require.NoError(t, err)
	require.Equal(t, "complete response", resp.Content)
	require.Equal(t, "test-cheap", resp.Model)
}