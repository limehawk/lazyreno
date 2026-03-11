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
			"renovateVersion": "43.55.4",
			"bootTime":        "2026-03-10 14:57:05",
			"app": map[string]any{
				"organizationCount": 1,
				"repositoryCount":   43,
			},
			"jobs": map[string]any{
				"queue": map[string]any{
					"size":       2,
					"inProgress": []any{},
				},
				"history": map[string]any{
					"processed": 43,
				},
			},
		})
	}))
	defer srv.Close()

	client := NewRenovateClient(srv.URL, "test-secret")
	status, err := client.GetStatus()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Version != "43.55.4" {
		t.Errorf("expected version 43.55.4, got %s", status.Version)
	}
	if status.QueueSize != 2 {
		t.Errorf("expected queue size 2, got %d", status.QueueSize)
	}
}

func TestRenovateJobQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/system/v1/jobs/queue" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"running": []map[string]any{
				{"jobId": "job-1", "repository": "limehawk/gruman-law-website"},
			},
			"pending": []map[string]any{
				{"jobId": "job-2", "repository": "limehawk/mill-mama-website"},
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
