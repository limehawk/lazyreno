package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type RenovateClient struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

func NewRenovateClient(baseURL, secret string) *RenovateClient {
	return &RenovateClient{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *RenovateClient) do(method, path string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	return c.httpClient.Do(req)
}

func (c *RenovateClient) GetStatus() (*SystemStatus, error) {
	resp, err := c.do("GET", "/system/v1/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		RenovateVersion string `json:"renovateVersion"`
		BootTime        string `json:"bootTime"`
		App             struct {
			OrganizationCount int `json:"organizationCount"`
			RepositoryCount   int `json:"repositoryCount"`
		} `json:"app"`
		Jobs struct {
			Queue struct {
				Size       int `json:"size"`
				InProgress []struct {
					JobID      string `json:"jobId"`
					Repository string `json:"repository"`
				} `json:"inProgress"`
			} `json:"queue"`
			History struct {
				Processed    int `json:"processed"`
				LastFinished struct {
					JobID      string `json:"jobId"`
					Repository string `json:"repository"`
					Status     string `json:"status"`
					Reason     string `json:"reason"`
					StartedAt  string `json:"startedAt"`
					FinishedAt string `json:"finishedAt"`
				} `json:"lastFinished"`
			} `json:"history"`
		} `json:"jobs"`
		Scheduler struct {
			Sync struct {
				Cron string `json:"cron"`
			} `json:"sync"`
		} `json:"scheduler"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	// Parse boot time — format is "2006-01-02 15:04:05"
	bootTime, _ := time.Parse("2006-01-02 15:04:05", raw.BootTime)

	status := &SystemStatus{
		Version:    raw.RenovateVersion,
		BootTime:   bootTime,
		Uptime:     time.Since(bootTime),
		QueueSize:  raw.Jobs.Queue.Size,
		RunningJob: len(raw.Jobs.Queue.InProgress),
		Enabled: map[string]bool{
			"api": true,
		},
	}

	// Parse last finished job if available.
	lf := raw.Jobs.History.LastFinished
	if lf.JobID != "" {
		job := Job{ID: lf.JobID, Repo: lf.Repository, Status: lf.Status, Trigger: lf.Reason}
		if t, err := time.Parse("2006-01-02 15:04:05", lf.StartedAt); err == nil {
			job.StartedAt = &t
		}
		if started, err1 := time.Parse("2006-01-02 15:04:05", lf.StartedAt); err1 == nil {
			if finished, err2 := time.Parse("2006-01-02 15:04:05", lf.FinishedAt); err2 == nil {
				job.Duration = finished.Sub(started)
			}
		}
		status.LastFinished = &job
	}

	return status, nil
}

func (c *RenovateClient) GetJobQueue() ([]Job, error) {
	resp, err := c.do("GET", "/system/v1/jobs/queue")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw struct {
		Running []struct {
			JobID      string `json:"jobId"`
			Repository string `json:"repository"`
		} `json:"running"`
		Pending []struct {
			JobID      string `json:"jobId"`
			Repository string `json:"repository"`
		} `json:"pending"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var jobs []Job
	for _, j := range raw.Running {
		jobs = append(jobs, Job{ID: j.JobID, Repo: j.Repository, Status: "running"})
	}
	for _, j := range raw.Pending {
		jobs = append(jobs, Job{ID: j.JobID, Repo: j.Repository, Status: "pending"})
	}
	return jobs, nil
}

func (c *RenovateClient) TriggerSync() error {
	resp, err := c.do("POST", "/system/v1/sync")
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sync failed: %s", resp.Status)
	}
	return nil
}

func (c *RenovateClient) PurgeFailedJobs() error {
	resp, err := c.do("POST", "/system/v1/jobs/purge")
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

