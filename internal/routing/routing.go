package routing

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/helloodokai/charter/internal/config"
	"github.com/helloodokai/charter/internal/models"
)

type Router struct {
	clients  map[models.Provider]models.Client
	profile  config.ProfileConfig
	fallback *config.ProfileConfig
}

func NewRouter(cfg *config.Config, profileName string) (*Router, error) {
	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		return nil, err
	}

	clients := make(map[models.Provider]models.Client)
	clients[models.ProviderOllamaCloud] = models.NewOllamaCloudClient(cfg.Models.OllamaCloud)
	clients[models.ProviderOllamaLocal] = models.NewOllamaLocalClient(cfg.Models.OllamaLocal)
	if cfg.Models.Anthropic.APIKey != "" {
		clients[models.ProviderAnthropic] = models.NewAnthropicClient(cfg.Models.Anthropic)
	}
	if cfg.Models.OpenAI.APIKey != "" {
		clients[models.ProviderOpenAI] = models.NewOpenAIClient(cfg.Models.OpenAI)
	}

	var fallback *config.ProfileConfig
	if cfg.Models.FallbackToLocal && profileName != "local" {
		if fp, ok := cfg.Models.Profiles["local"]; ok {
			fallback = &fp
		}
	}

	return &Router{
		clients:  clients,
		profile:  profile,
		fallback: fallback,
	}, nil
}

func (r *Router) Complete(ctx context.Context, tier models.Tier, req models.CompletionRequest) (*models.CompletionResponse, error) {
	ref := r.tierRef(tier)
	resp, err := r.completeWithRef(ctx, ref, req)
	if err != nil && r.fallback != nil {
		slog.Warn("primary provider failed, trying fallback", "tier", tier, "error", err)
		fbRef := r.fallbackTierRef(tier)
		return r.completeWithRef(ctx, fbRef, req)
	}
	return resp, err
}

func (r *Router) Stream(ctx context.Context, tier models.Tier, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
	ref := r.tierRef(tier)
	resp, err := r.streamWithRef(ctx, ref, req, w)
	if err != nil && r.fallback != nil {
		slog.Warn("primary provider failed, trying fallback", "tier", tier, "error", err)
		fbRef := r.fallbackTierRef(tier)
		return r.streamWithRef(ctx, fbRef, req, w)
	}
	return resp, err
}

func (r *Router) completeWithRef(ctx context.Context, ref config.ModelRef, req models.CompletionRequest) (*models.CompletionResponse, error) {
	client, ok := r.clients[models.Provider(ref.Provider)]
	if !ok {
		return nil, fmt.Errorf("no client for provider %q", ref.Provider)
	}
	req.Model = ref.Name
	slog.Debug("routing completion", "provider", ref.Provider, "model", ref.Name, "tier", req.Model)
	return client.Complete(ctx, req)
}

func (r *Router) streamWithRef(ctx context.Context, ref config.ModelRef, req models.CompletionRequest, w io.Writer) (*models.CompletionResponse, error) {
	client, ok := r.clients[models.Provider(ref.Provider)]
	if !ok {
		return nil, fmt.Errorf("no client for provider %q", ref.Provider)
	}
	req.Model = ref.Name
	slog.Debug("routing stream", "provider", ref.Provider, "model", ref.Name)
	return client.Stream(ctx, req, w)
}

func (r *Router) tierRef(tier models.Tier) config.ModelRef {
	switch tier {
	case models.Cheap:
		return r.profile.Cheap
	case models.Mid:
		return r.profile.Mid
	case models.Frontier:
		return r.profile.Frontier
	default:
		return r.profile.Mid
	}
}

func (r *Router) fallbackTierRef(tier models.Tier) config.ModelRef {
	if r.fallback == nil {
		return r.tierRef(tier)
	}
	switch tier {
	case models.Cheap:
		return r.fallback.Cheap
	case models.Mid:
		return r.fallback.Mid
	case models.Frontier:
		return r.fallback.Frontier
	default:
		return r.fallback.Mid
	}
}

func (r *Router) CheckReachable(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for prov, client := range r.clients {
		_, err := client.Complete(ctx, models.CompletionRequest{
			Messages: []models.Message{{Role: "user", Content: "ping"}},
			MaxTokens: 1,
		})
		if err != nil {
			results[string(prov)] = err
		}
	}
	return results
}