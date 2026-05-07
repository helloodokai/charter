package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAndLoad_repoLevelCharterToml(t *testing.T) {
	tmp := t.TempDir()

	// Create repo-level .charter.toml with a custom value
	repoCfg := filepath.Join(tmp, ".charter.toml")
	if err := os.WriteFile(repoCfg, []byte(`
[storage]
charters_dir = "repo-charters"
`), 0644); err != nil {
		t.Fatalf("writing repo config: %v", err)
	}

	cfg, err := FindAndLoad(tmp)
	if err != nil {
		t.Fatalf("FindAndLoad error: %v", err)
	}

	if cfg.Storage.ChartersDir != "repo-charters" {
		t.Errorf("expected charters_dir = repo-charters, got %q", cfg.Storage.ChartersDir)
	}
}

func TestFindAndLoad_repoLevelFallbackCharterToml(t *testing.T) {
	tmp := t.TempDir()

	// Create repo-level charter.toml (without the dot)
	repoCfg := filepath.Join(tmp, "charter.toml")
	if err := os.WriteFile(repoCfg, []byte(`
[storage]
charters_dir = "fallback-charters"
`), 0644); err != nil {
		t.Fatalf("writing repo config: %v", err)
	}

	cfg, err := FindAndLoad(tmp)
	if err != nil {
		t.Fatalf("FindAndLoad error: %v", err)
	}

	if cfg.Storage.ChartersDir != "fallback-charters" {
		t.Errorf("expected charters_dir = fallback-charters, got %q", cfg.Storage.ChartersDir)
	}
}

func TestFindAndLoad_userLevelCharterToml(t *testing.T) {
	tmp := t.TempDir()

	// Create a fake home directory with ~/.charter/.charter.toml
	charterDir := filepath.Join(tmp, ".charter")
	if err := os.MkdirAll(charterDir, 0750); err != nil {
		t.Fatalf("creating .charter dir: %v", err)
	}
	userCfg := filepath.Join(charterDir, ".charter.toml")
	if err := os.WriteFile(userCfg, []byte(`
[storage]
charters_dir = "user-charters"
`), 0644); err != nil {
		t.Fatalf("writing user config: %v", err)
	}

	// Override HOME so os.UserHomeDir() returns our temp directory
	t.Setenv("HOME", tmp)

	cfg, err := FindAndLoad(tmp)
	if err != nil {
		t.Fatalf("FindAndLoad error: %v", err)
	}

	if cfg.Storage.ChartersDir != "user-charters" {
		t.Errorf("expected charters_dir = user-charters, got %q", cfg.Storage.ChartersDir)
	}
}

func TestFindAndLoad_defaultsWhenMissing(t *testing.T) {
	tmp := t.TempDir()

	// Override HOME to an empty temp dir so no user-level config exists
	t.Setenv("HOME", tmp)

	cfg, err := FindAndLoad(tmp)
	if err != nil {
		t.Fatalf("FindAndLoad error: %v", err)
	}

	if cfg.Storage.ChartersDir != ".charters" {
		t.Errorf("expected default charters_dir = .charters, got %q", cfg.Storage.ChartersDir)
	}
}

func TestFindAndLoad_repoOverridesUser(t *testing.T) {
	repoTmp := t.TempDir()
	userTmp := t.TempDir()

	// Create repo-level config
	repoCfg := filepath.Join(repoTmp, ".charter.toml")
	if err := os.WriteFile(repoCfg, []byte(`
[storage]
charters_dir = "repo-charters"
`), 0644); err != nil {
		t.Fatalf("writing repo config: %v", err)
	}

	// Create user-level config with different value
	charterDir := filepath.Join(userTmp, ".charter")
	if err := os.MkdirAll(charterDir, 0750); err != nil {
		t.Fatalf("creating .charter dir: %v", err)
	}
	userCfg := filepath.Join(charterDir, ".charter.toml")
	if err := os.WriteFile(userCfg, []byte(`
[storage]
charters_dir = "user-charters"
`), 0644); err != nil {
		t.Fatalf("writing user config: %v", err)
	}

	// Override HOME so user-level config is visible
	t.Setenv("HOME", userTmp)

	cfg, err := FindAndLoad(repoTmp)
	if err != nil {
		t.Fatalf("FindAndLoad error: %v", err)
	}

	if cfg.Storage.ChartersDir != "repo-charters" {
		t.Errorf("expected repo config to override user config: want charters_dir = repo-charters, got %q", cfg.Storage.ChartersDir)
	}
}
