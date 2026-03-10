package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Renovate RenovateConfig `toml:"renovate"`
	GitHub   GitHubConfig   `toml:"github"`
	UI       UIConfig       `toml:"ui"`
}

type RenovateConfig struct {
	URL    string `toml:"url"`
	Secret string `toml:"-"` // env only
}

type GitHubConfig struct {
	Owner string `toml:"owner"`
	Token string `toml:"-"` // env only
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

	return cfg, nil
}
