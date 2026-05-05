package githubapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/go-github/v66/github"
)

type Server struct {
	port       int
	appID      int64
	privateKey []byte
	handler    *http.ServeMux
}

type Config struct {
	Port       int
	AppID      int64
	PrivateKey []byte
}

func NewServer(cfg Config) (*Server, error) {
	s := &Server{
		port:       cfg.Port,
		appID:      cfg.AppID,
		privateKey: cfg.PrivateKey,
		handler:    http.NewServeMux(),
	}

	s.handler.HandleFunc("/webhook", s.handleWebhook)
	s.handler.HandleFunc("/health", s.handleHealth)

	return s, nil
}

func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.handler,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("charter app server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
	}

	slog.Info("shutting down server")
	return srv.Shutdown(ctx)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	_ = r.Context()

	payload, err := github.ValidatePayload(r, nil)
	if err != nil {
		slog.Error("validating payload", "error", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		slog.Error("parsing webhook", "error", err)
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}

	switch e := event.(type) {
	case *github.IssuesEvent:
		if err := s.handleIssueEvent(r.Context(), e); err != nil {
			slog.Error("handling issue event", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	case *github.IssueCommentEvent:
		if err := s.handleCommentEvent(r.Context(), e); err != nil {
			slog.Error("handling comment event", "error", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	default:
		slog.Debug("ignoring event", "type", fmt.Sprintf("%T", event))
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleIssueEvent(ctx context.Context, e *github.IssuesEvent) error {
	if e.GetAction() != "labeled" {
		return nil
	}

	label := e.GetLabel().GetName()
	if label != "needs-charter" {
		return nil
	}

	slog.Info("issue labeled needs-charter",
		"repo", e.GetRepo().GetFullName(),
		"issue", e.GetIssue().GetNumber(),
	)

	return nil
}

func (s *Server) handleCommentEvent(ctx context.Context, e *github.IssueCommentEvent) error {
	if e.GetAction() != "created" {
		return nil
	}

	slog.Info("issue comment",
		"repo", e.GetRepo().GetFullName(),
		"issue", e.GetIssue().GetNumber(),
	)
	return nil
}