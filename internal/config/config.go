package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config holds the top-level charter configuration.
type Config struct {
	Dialogue DialogueConfig `toml:"dialogue"`
	Storage  StorageConfig  `toml:"storage"`
	Models   ModelsConfig   `toml:"models"`
	GitHub   GitHubConfig  `toml:"github"`
	Paths    PathsConfig    `toml:"paths"`
}

// DialogueConfig configures the dialogue turn behaviour.
type DialogueConfig struct {
	TurnBudget      int    `toml:"turn_budget"`
	AskForRollback  string `toml:"ask_for_rollback_at"`
	RequireCounterSpec bool  `toml:"require_counter_spec"`
}

// StorageConfig configures charter storage paths.
type StorageConfig struct {
	ChartersDir string `toml:"charters_dir"`
}

// ModelsConfig configures LLM provider connections and profiles.
type ModelsConfig struct {
	DefaultProfile   string                    `toml:"default_profile"`
	FallbackToLocal  bool                      `toml:"fallback_to_local"`
	Profiles         map[string]ProfileConfig  `toml:"profiles"`
	OllamaCloud      OllamaConfig              `toml:"ollama_cloud"`
	OllamaLocal      OllamaConfig              `toml:"ollama_local"`
	Anthropic        AnthropicConfig            `toml:"anthropic"`
	OpenAI           OpenAIConfig               `toml:"openai"`
}

// ProfileConfig maps each tier to a model reference for a routing profile.
type ProfileConfig struct {
	Cheap      ModelRef `toml:"cheap"`
	Mid        ModelRef `toml:"mid"`
	Frontier   ModelRef `toml:"frontier"`
	WebSearch  bool     `toml:"web_search"`
}

// ModelRef references a specific model by provider and name.
type ModelRef struct {
	Provider string `toml:"provider"`
	Name     string `toml:"name"`
}

// OllamaConfig configures an Ollama host connection.
type OllamaConfig struct {
	Host   string `toml:"host"`
	APIKey string `toml:"api_key"`
}

// AnthropicConfig configures the Anthropic API connection.
type AnthropicConfig struct {
	APIKey string `toml:"api_key"`
}

// OpenAIConfig configures the OpenAI API connection.
type OpenAIConfig struct {
	APIKey string `toml:"api_key"`
}

// GitHubConfig configures GitHub App authentication and label settings.
type GitHubConfig struct {
	AppIDEnv      string `toml:"app_id_env"`
	PrivateKeyEnv string `toml:"private_key_env"`
	NeedsLabel    string `toml:"needs_label"`
	HasLabel      string `toml:"has_label"`
}

// PathsConfig lists critical path globs that influence risk assessment.
type PathsConfig struct {
	Critical []string `toml:"critical"`
}

// Default returns a Config populated with sensible defaults.
func Default() *Config {
	return &Config{
		Dialogue: DialogueConfig{
			TurnBudget:      8,
			AskForRollback:  "high",
			RequireCounterSpec: true,
		},
		Storage: StorageConfig{
			ChartersDir: ".charters",
		},
		Models: ModelsConfig{
			DefaultProfile:  "cloud",
			FallbackToLocal: true,
			Profiles: map[string]ProfileConfig{
				"cloud": {
					Cheap:      ModelRef{Provider: "ollama_cloud", Name: "gpt-oss:20b"},
					Mid:        ModelRef{Provider: "ollama_cloud", Name: "qwen3-coder:480b"},
					Frontier:   ModelRef{Provider: "anthropic", Name: "claude-sonnet-4-6"},
					WebSearch:  true,
				},
				"local": {
					Cheap:     ModelRef{Provider: "ollama_local", Name: "gemma3:12b"},
					Mid:       ModelRef{Provider: "ollama_local", Name: "gemma3:12b"},
					Frontier:  ModelRef{Provider: "ollama_local", Name: "gemma3:12b"},
					WebSearch: true,
				},
			},
		OllamaCloud: OllamaConfig{ //nolint:gosec // false positive: env var template, not hardcoded credential
			Host:   "https://ollama.com",
			APIKey: "${OLLAMA_API_KEY}",
		},
		OllamaLocal: OllamaConfig{
			Host: "http://localhost:11434",
		},
		Anthropic: AnthropicConfig{ //nolint:gosec // false positive: env var template, not hardcoded credential
			APIKey: "${ANTHROPIC_API_KEY}",
		},
		OpenAI: OpenAIConfig{ //nolint:gosec // false positive: env var template, not hardcoded credential
			APIKey: "${OPENAI_API_KEY}",
		},
		},
		GitHub: GitHubConfig{
			AppIDEnv:      "GITHUB_APP_ID",
			PrivateKeyEnv: "GITHUB_APP_PRIVATE_KEY",
			NeedsLabel:    "needs-charter",
			HasLabel:      "has-charter",
		},
		Paths: PathsConfig{
			Critical: []string{"src/auth/**", "src/payments/**", "migrations/**"},
		},
	}
}

// Load reads and parses a TOML config file, falling back to defaults when the file is missing.
func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path) //nolint:gosec // expected: user-specified config path
	if err != nil {
		if os.IsNotExist(err) {
			cfg.expandEnv()
			cfg.resolveAPIKeys()
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	cfg.expandEnv()
	cfg.resolveAPIKeys()
	return cfg, nil
}

// FindAndLoad searches standard paths for a config file and loads it, or returns defaults.
// It checks in order: repo-level .charter.toml, repo-level charter.toml,
// then user-level ~/.charter/.charter.toml.
func FindAndLoad(repoRoot string) (*Config, error) {
	paths := []string{
		filepath.Join(repoRoot, ".charter.toml"),
		filepath.Join(repoRoot, "charter.toml"),
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".charter", ".charter.toml"))
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return Load(p)
		}
	}
	cfg := Default()
	cfg.expandEnv()
	cfg.resolveAPIKeys()
	return cfg, nil
}

func (c *Config) expandEnv() {
	c.Models.OllamaCloud.APIKey = os.ExpandEnv(c.Models.OllamaCloud.APIKey)
	c.Models.Anthropic.APIKey = os.ExpandEnv(c.Models.Anthropic.APIKey)
	c.Models.OpenAI.APIKey = os.ExpandEnv(c.Models.OpenAI.APIKey)
	c.Models.OllamaCloud.Host = os.ExpandEnv(c.Models.OllamaCloud.Host)
	c.Models.OllamaLocal.Host = os.ExpandEnv(c.Models.OllamaLocal.Host)
}

func (c *Config) resolveAPIKeys() {
	if c.Models.OllamaCloud.APIKey == "" {
		c.Models.OllamaCloud.APIKey = os.Getenv("OLLAMA_API_KEY")
	}
	if c.Models.Anthropic.APIKey == "" {
		c.Models.Anthropic.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if c.Models.OpenAI.APIKey == "" {
		c.Models.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
	}
}

// GetProfile returns the ProfileConfig for the named profile, or an error if not found.
func (c *Config) GetProfile(name string) (ProfileConfig, error) {
	p, ok := c.Models.Profiles[name]
	if !ok {
		return ProfileConfig{}, fmt.Errorf("unknown profile %q", name)
	}
	return p, nil
}

// ChartersDir returns the full path to the charters directory under the given repo root.
func (c *Config) ChartersDir(repoRoot string) string {
	return filepath.Join(repoRoot, c.Storage.ChartersDir)
}

// MarshalTOML serialises the Config to TOML bytes.
func MarshalTOML(c *Config) ([]byte, error) {
	var buf strings.Builder
	bufPtr := &buf

	buf.WriteString("# Charter configuration\n")
	buf.WriteString("# See https://github.com/helloodokai/charter for full docs\n")
	buf.WriteString("\n")

	buf.WriteString("[dialogue]\n")
	fmt.Fprintf(bufPtr, "turn_budget         = %d\n", c.Dialogue.TurnBudget)
	fmt.Fprintf(bufPtr, "ask_for_rollback_at = %q\n", c.Dialogue.AskForRollback)
	fmt.Fprintf(bufPtr, "require_counter_spec = %v\n", c.Dialogue.RequireCounterSpec)
	buf.WriteString("\n")

	buf.WriteString("[storage]\n")
	fmt.Fprintf(bufPtr, "charters_dir = %q\n", c.Storage.ChartersDir)
	buf.WriteString("\n")

	buf.WriteString("[models]\n")
	fmt.Fprintf(bufPtr, "default_profile    = %q\n", c.Models.DefaultProfile)
	fmt.Fprintf(bufPtr, "fallback_to_local  = %v\n", c.Models.FallbackToLocal)
	buf.WriteString("\n")

	writeProfile := func(name string, p ProfileConfig) {
		fmt.Fprintf(bufPtr, "[models.profiles.%s]\n", name)
		fmt.Fprintf(bufPtr, "cheap      = { provider = %q, name = %q }\n", p.Cheap.Provider, p.Cheap.Name)
		fmt.Fprintf(bufPtr, "mid        = { provider = %q, name = %q }\n", p.Mid.Provider, p.Mid.Name)
		fmt.Fprintf(bufPtr, "frontier   = { provider = %q, name = %q }\n", p.Frontier.Provider, p.Frontier.Name)
		fmt.Fprintf(bufPtr, "web_search = %v\n", p.WebSearch)
		buf.WriteString("\n")
	}
	writeProfile("cloud", c.Models.Profiles["cloud"])
	writeProfile("local", c.Models.Profiles["local"])

	buf.WriteString("[models.ollama_cloud]\n")
	fmt.Fprintf(bufPtr, "host     = %q\n", c.Models.OllamaCloud.Host)
	buf.WriteString("api_key  = \"${OLLAMA_API_KEY}\"\n")
	buf.WriteString("\n")

	buf.WriteString("[models.ollama_local]\n")
	fmt.Fprintf(bufPtr, "host     = %q\n", c.Models.OllamaLocal.Host)
	buf.WriteString("\n")

	buf.WriteString("[models.anthropic]\n")
	buf.WriteString("api_key = \"${ANTHROPIC_API_KEY}\"\n")
	buf.WriteString("\n")

	buf.WriteString("[models.openai]\n")
	buf.WriteString("api_key = \"${OPENAI_API_KEY}\"\n")
	buf.WriteString("\n")

	buf.WriteString("[github]\n")
	fmt.Fprintf(bufPtr, "app_id_env       = %q\n", c.GitHub.AppIDEnv)
	fmt.Fprintf(bufPtr, "private_key_env  = %q\n", c.GitHub.PrivateKeyEnv)
	fmt.Fprintf(bufPtr, "needs_label      = %q\n", c.GitHub.NeedsLabel)
	fmt.Fprintf(bufPtr, "has_label        = %q\n", c.GitHub.HasLabel)
	buf.WriteString("\n")

	buf.WriteString("[paths]\n")
	buf.WriteString("critical = [")
	for i, p := range c.Paths.Critical {
		if i > 0 {
			buf.WriteString(", ")
		}
		fmt.Fprintf(bufPtr, "%q", p)
	}
	buf.WriteString("]\n")

	return []byte(buf.String()), nil
}