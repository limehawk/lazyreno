package config

import (
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Renovate RenovateConfig `toml:"renovate"`
	GitHub   GitHubConfig   `toml:"github"`
	UI       UIConfig       `toml:"ui"`
}

type RenovateConfig struct {
	URL    string `toml:"url"`
	Secret string `toml:"secret"`
}

type GitHubConfig struct {
	Owner string `toml:"owner"`
	Token string `toml:"token"`
}

type UIConfig struct {
	Accent          string `toml:"accent"`
	RefreshInterval string `toml:"refresh_interval"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		UI: UIConfig{
			Accent:          "cyan",
			RefreshInterval: "30s",
		},
	}

	if path != "" {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, err
		}
	}

	// Env var overrides
	if v := os.Getenv("LAZYRENO_RENOVATE_SECRET"); v != "" {
		cfg.Renovate.Secret = v
	}
	if v := os.Getenv("LAZYRENO_RENOVATE_URL"); v != "" {
		cfg.Renovate.URL = v
	}
	if v := os.Getenv("LAZYRENO_GITHUB_TOKEN"); v != "" {
		cfg.GitHub.Token = v
	} else if v := os.Getenv("GITHUB_TOKEN"); v != "" {
		cfg.GitHub.Token = v
	}
	if v := os.Getenv("LAZYRENO_GITHUB_OWNER"); v != "" {
		cfg.GitHub.Owner = v
	}

	// Resolve op:// secret references via 1Password CLI
	cfg.Renovate.Secret = resolveSecret(cfg.Renovate.Secret)
	cfg.GitHub.Token = resolveSecret(cfg.GitHub.Token)

	return cfg, nil
}

// resolveSecret resolves an op:// reference via the 1Password CLI.
// Returns the original value if it's not an op:// reference or if resolution fails.
func resolveSecret(val string) string {
	if !strings.HasPrefix(val, "op://") {
		return val
	}
	out, err := exec.Command("op", "read", val).Output()
	if err != nil {
		return val
	}
	return strings.TrimSpace(string(out))
}
