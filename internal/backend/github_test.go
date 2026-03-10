package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubListPRs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/repos/limehawk/test-repo/pulls" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]any{
			{
				"number":     1,
				"title":      "chore(deps): update shadcn to v4",
				"html_url":   "https://github.com/limehawk/test-repo/pull/1",
				"state":      "open",
				"head":       map[string]any{"ref": "renovate/shadcn-4.x"},
				"base":       map[string]any{"ref": "main"},
				"labels":     []map[string]any{{"name": "renovate/minor"}},
				"created_at": "2026-03-08T00:00:00Z",
			},
		})
	}))
	defer srv.Close()

	client := NewGitHubClient("test-token", srv.URL)
	prs, err := client.ListOpenPRs("limehawk", "test-repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(prs))
	}
	if prs[0].Title != "chore(deps): update shadcn to v4" {
		t.Errorf("unexpected title: %s", prs[0].Title)
	}
	if prs[0].Branch != "renovate/shadcn-4.x" {
		t.Errorf("unexpected branch: %s", prs[0].Branch)
	}
}

func TestGitHubMergePR(t *testing.T) {
	merged := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v3/repos/limehawk/test-repo/pulls/1/merge" && r.Method == "PUT":
			merged = true
			json.NewEncoder(w).Encode(map[string]any{"merged": true})
		}
	}))
	defer srv.Close()

	client := NewGitHubClient("test-token", srv.URL)
	err := client.MergePR("limehawk", "test-repo", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !merged {
		t.Error("merge endpoint not called")
	}
}

func TestGitHubClosePR(t *testing.T) {
	closed := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/repos/limehawk/test-repo/pulls/1" && r.Method == "PATCH" {
			closed = true
			json.NewEncoder(w).Encode(map[string]any{"state": "closed"})
		}
	}))
	defer srv.Close()

	client := NewGitHubClient("test-token", srv.URL)
	err := client.ClosePR("limehawk", "test-repo", 1, "renovate/test-branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closed {
		t.Error("close endpoint not called")
	}
}
