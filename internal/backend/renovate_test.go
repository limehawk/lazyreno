package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRenovateStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/system/v1/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Errorf("missing auth header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"version":  "14.1.0",
			"bootTime": "2026-03-10T00:00:00Z",
			"enabled": map[string]bool{
				"api":       true,
				"system":    true,
				"reporting": true,
				"jobs":      true,
			},
		})
	}))
	defer srv.Close()

	client := NewRenovateClient(srv.URL, "test-secret")
	status, err := client.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Version != "14.1.0" {
		t.Errorf("expected version 14.1.0, got %s", status.Version)
	}
}

func TestRenovateJobQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/system/v1/jobs/queue" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"running": []map[string]any{
				{"id": "job-1", "repository": "limehawk/gruman-law-website"},
			},
			"pending": []map[string]any{
				{"id": "job-2", "repository": "limehawk/mill-mama-website"},
			},
		})
	}))
	defer srv.Close()

	client := NewRenovateClient(srv.URL, "test-secret")
	jobs, err := client.GetJobQueue()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
	if jobs[0].Status != "running" {
		t.Errorf("expected first job running, got %s", jobs[0].Status)
	}
}

func TestRenovateTriggerSync(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/system/v1/sync" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewRenovateClient(srv.URL, "test-secret")
	err := client.TriggerSync()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("sync endpoint not called")
	}
}

func TestRenovateOrgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/orgs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "limehawk"},
		})
	}))
	defer srv.Close()

	client := NewRenovateClient(srv.URL, "test-secret")
	orgs, err := client.GetOrgs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orgs) != 1 || orgs[0] != "limehawk" {
		t.Errorf("expected [limehawk], got %v", orgs)
	}
}
