package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.UI.RefreshInterval != "30s" {
		t.Errorf("expected default refresh_interval 30s, got %s", cfg.UI.RefreshInterval)
	}
	if cfg.UI.Accent != "cyan" {
		t.Errorf("expected default accent cyan, got %s", cfg.UI.Accent)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	os.WriteFile(path, []byte(`
[renovate]
url = "https://reno.example.com"

[github]
owner = "testorg"

[ui]
refresh_interval = "60s"
`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Renovate.URL != "https://reno.example.com" {
		t.Errorf("expected url from file, got %s", cfg.Renovate.URL)
	}
	if cfg.GitHub.Owner != "testorg" {
		t.Errorf("expected owner from file, got %s", cfg.GitHub.Owner)
	}
	if cfg.UI.RefreshInterval != "60s" {
		t.Errorf("expected refresh_interval from file, got %s", cfg.UI.RefreshInterval)
	}
}

func TestEnvVarOverrides(t *testing.T) {
	t.Setenv("LAZYRENO_RENOVATE_SECRET", "test-secret")
	t.Setenv("LAZYRENO_GITHUB_TOKEN", "test-gh-token")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Renovate.Secret != "test-secret" {
		t.Errorf("expected secret from env, got %s", cfg.Renovate.Secret)
	}
	if cfg.GitHub.Token != "test-gh-token" {
		t.Errorf("expected token from env, got %s", cfg.GitHub.Token)
	}
}

func TestGitHubTokenFallback(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "fallback-token")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHub.Token != "fallback-token" {
		t.Errorf("expected GITHUB_TOKEN fallback, got %s", cfg.GitHub.Token)
	}
}
