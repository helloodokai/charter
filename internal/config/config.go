package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Dialogue DialogueConfig `toml:"dialogue"`
	Storage  StorageConfig  `toml:"storage"`
	Models   ModelsConfig   `toml:"models"`
	GitHub   GitHubConfig  `toml:"github"`
	Paths    PathsConfig    `toml:"paths"`
}

type DialogueConfig struct {
	TurnBudget      int    `toml:"turn_budget"`
	AskForRollback  string `toml:"ask_for_rollback_at"`
	RequireCounterSpec bool  `toml:"require_counter_spec"`
}

type StorageConfig struct {
	ChartersDir string `toml:"charters_dir"`
}

type ModelsConfig struct {
	DefaultProfile   string                    `toml:"default_profile"`
	FallbackToLocal  bool                      `toml:"fallback_to_local"`
	Profiles         map[string]ProfileConfig  `toml:"profiles"`
	OllamaCloud      OllamaConfig              `toml:"ollama_cloud"`
	OllamaLocal      OllamaConfig              `toml:"ollama_local"`
	Anthropic        AnthropicConfig            `toml:"anthropic"`
	OpenAI           OpenAIConfig               `toml:"openai"`
}

type ProfileConfig struct {
	Cheap    ModelRef `toml:"cheap"`
	Mid      ModelRef `toml:"mid"`
	Frontier ModelRef `toml:"frontier"`
}

type ModelRef struct {
	Provider string `toml:"provider"`
	Name     string `toml:"name"`
}

type OllamaConfig struct {
	Host   string `toml:"host"`
	APIKey string `toml:"api_key"`
}

type AnthropicConfig struct {
	APIKey string `toml:"api_key"`
}

type OpenAIConfig struct {
	APIKey string `toml:"api_key"`
}

type GitHubConfig struct {
	AppIDEnv      string `toml:"app_id_env"`
	PrivateKeyEnv string `toml:"private_key_env"`
	NeedsLabel    string `toml:"needs_label"`
	HasLabel      string `toml:"has_label"`
}

type PathsConfig struct {
	Critical []string `toml:"critical"`
}

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
					Cheap:    ModelRef{Provider: "ollama_cloud", Name: "gpt-oss:20b"},
					Mid:      ModelRef{Provider: "ollama_cloud", Name: "qwen3-coder:480b"},
					Frontier: ModelRef{Provider: "anthropic", Name: "claude-sonnet-4-6"},
				},
				"local": {
					Cheap:    ModelRef{Provider: "ollama_local", Name: "qwen2.5-coder:7b"},
					Mid:      ModelRef{Provider: "ollama_local", Name: "qwen2.5-coder:32b"},
					Frontier: ModelRef{Provider: "anthropic", Name: "claude-sonnet-4-6"},
				},
			},
			OllamaCloud: OllamaConfig{
				Host:   "https://ollama.com",
				APIKey: "${OLLAMA_API_KEY}",
			},
			OllamaLocal: OllamaConfig{
				Host: "http://localhost:11434",
			},
			Anthropic: AnthropicConfig{
				APIKey: "${ANTHROPIC_API_KEY}",
			},
			OpenAI: OpenAIConfig{
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

func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
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

func FindAndLoad(repoRoot string) (*Config, error) {
	paths := []string{
		filepath.Join(repoRoot, ".charter.toml"),
		filepath.Join(repoRoot, "charter.toml"),
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

func (c *Config) GetProfile(name string) (ProfileConfig, error) {
	p, ok := c.Models.Profiles[name]
	if !ok {
		return ProfileConfig{}, fmt.Errorf("unknown profile %q", name)
	}
	return p, nil
}

func (c *Config) ChartersDir(repoRoot string) string {
	return filepath.Join(repoRoot, c.Storage.ChartersDir)
}

func MarshalTOML(c *Config) ([]byte, error) {
	var buf strings.Builder

	buf.WriteString("# Charter configuration\n")
	buf.WriteString("# See https://github.com/helloodokai/charter for full docs\n")
	buf.WriteString("\n")

	buf.WriteString("[dialogue]\n")
	buf.WriteString(fmt.Sprintf("turn_budget         = %d\n", c.Dialogue.TurnBudget))
	buf.WriteString(fmt.Sprintf("ask_for_rollback_at = %q\n", c.Dialogue.AskForRollback))
	buf.WriteString(fmt.Sprintf("require_counter_spec = %v\n", c.Dialogue.RequireCounterSpec))
	buf.WriteString("\n")

	buf.WriteString("[storage]\n")
	buf.WriteString(fmt.Sprintf("charters_dir = %q\n", c.Storage.ChartersDir))
	buf.WriteString("\n")

	buf.WriteString("[models]\n")
	buf.WriteString(fmt.Sprintf("default_profile    = %q\n", c.Models.DefaultProfile))
	buf.WriteString(fmt.Sprintf("fallback_to_local  = %v\n", c.Models.FallbackToLocal))
	buf.WriteString("\n")

	writeProfile := func(name string, p ProfileConfig) {
		buf.WriteString(fmt.Sprintf("[models.profiles.%s]\n", name))
		buf.WriteString(fmt.Sprintf("cheap    = { provider = %q, name = %q }\n", p.Cheap.Provider, p.Cheap.Name))
		buf.WriteString(fmt.Sprintf("mid      = { provider = %q, name = %q }\n", p.Mid.Provider, p.Mid.Name))
		buf.WriteString(fmt.Sprintf("frontier = { provider = %q, name = %q }\n", p.Frontier.Provider, p.Frontier.Name))
		buf.WriteString("\n")
	}
	writeProfile("cloud", c.Models.Profiles["cloud"])
	writeProfile("local", c.Models.Profiles["local"])

	buf.WriteString("[models.ollama_cloud]\n")
	buf.WriteString(fmt.Sprintf("host     = %q\n", c.Models.OllamaCloud.Host))
	buf.WriteString("api_key  = \"${OLLAMA_API_KEY}\"\n")
	buf.WriteString("\n")

	buf.WriteString("[models.ollama_local]\n")
	buf.WriteString(fmt.Sprintf("host     = %q\n", c.Models.OllamaLocal.Host))
	buf.WriteString("\n")

	buf.WriteString("[models.anthropic]\n")
	buf.WriteString("api_key = \"${ANTHROPIC_API_KEY}\"\n")
	buf.WriteString("\n")

	buf.WriteString("[models.openai]\n")
	buf.WriteString("api_key = \"${OPENAI_API_KEY}\"\n")
	buf.WriteString("\n")

	buf.WriteString("[github]\n")
	buf.WriteString(fmt.Sprintf("app_id_env       = %q\n", c.GitHub.AppIDEnv))
	buf.WriteString(fmt.Sprintf("private_key_env  = %q\n", c.GitHub.PrivateKeyEnv))
	buf.WriteString(fmt.Sprintf("needs_label      = %q\n", c.GitHub.NeedsLabel))
	buf.WriteString(fmt.Sprintf("has_label        = %q\n", c.GitHub.HasLabel))
	buf.WriteString("\n")

	buf.WriteString("[paths]\n")
	buf.WriteString("critical = [")
	for i, p := range c.Paths.Critical {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%q", p))
	}
	buf.WriteString("]\n")

	return []byte(buf.String()), nil
}