use std::time::Duration;

use anyhow::Result;
use chrono::NaiveDateTime;
use reqwest::Client;
use serde::Deserialize;

use crate::types::{Job, JobState, SystemStatus};

// ---------------------------------------------------------------------------
// Response types — matches the actual Renovate CE API JSON structure
// ---------------------------------------------------------------------------

/// GET /system/v1/status
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StatusResponse {
    pub renovate_version: String,
    pub boot_time: String, // "2006-01-02 15:04:05" format
    pub jobs: StatusJobs,
}

#[derive(Debug, Deserialize)]
pub struct StatusJobs {
    pub queue: StatusQueue,
    pub history: StatusHistory,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StatusQueue {
    pub size: u64,
    pub in_progress: Vec<InProgressJob>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
pub struct InProgressJob {
    pub job_id: String,
    pub repository: String,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
#[allow(dead_code)]
pub struct StatusHistory {
    pub processed: u64,
    pub last_finished: Option<LastFinishedJob>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LastFinishedJob {
    pub job_id: String,
    pub repository: String,
    pub status: String,
    pub reason: Option<String>,
    pub started_at: Option<String>,
    pub finished_at: Option<String>,
}

/// GET /system/v1/jobs/queue
#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct JobQueueResponse {
    #[serde(default)]
    pub running: Vec<QueueJob>,
    #[serde(default)]
    pub pending: Vec<QueueJob>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct QueueJob {
    pub job_id: String,
    pub repository: String,
}

// ---------------------------------------------------------------------------
// Conversion to domain types
// ---------------------------------------------------------------------------

const TIME_FMT: &str = "%Y-%m-%d %H:%M:%S";

impl StatusResponse {
    pub fn into_status(self) -> SystemStatus {
        let boot_time = NaiveDateTime::parse_from_str(&self.boot_time, TIME_FMT)
            .ok()
            .and_then(|dt| dt.and_utc().into());
        let uptime = boot_time
            .map(|bt: chrono::DateTime<chrono::Utc>| {
                let dur = chrono::Utc::now() - bt;
                let days = dur.num_days();
                let hours = dur.num_hours() % 24;
                let mins = dur.num_minutes() % 60;
                if days > 0 {
                    format!("{days}d {hours}h")
                } else if hours > 0 {
                    format!("{hours}h {mins}m")
                } else {
                    format!("{mins}m")
                }
            })
            .unwrap_or_else(|| "unknown".into());

        let last_finished = self.jobs.history.last_finished.and_then(|lf| {
            if lf.job_id.is_empty() {
                return None;
            }
            let started = lf
                .started_at
                .as_deref()
                .and_then(|s| NaiveDateTime::parse_from_str(s, TIME_FMT).ok())
                .map(|dt| dt.and_utc());
            let finished = lf
                .finished_at
                .as_deref()
                .and_then(|s| NaiveDateTime::parse_from_str(s, TIME_FMT).ok())
                .map(|dt| dt.and_utc());
            let duration = match (started, finished) {
                (Some(s), Some(f)) => {
                    let d = f - s;
                    Some(Duration::from_secs(d.num_seconds().unsigned_abs()))
                }
                _ => None,
            };
            Some(Job {
                id: lf.job_id,
                repo: lf.repository,
                state: match lf.status.as_str() {
                    "finished" => JobState::Finished,
                    "failed" => JobState::Failed,
                    _ => JobState::Finished,
                },
                started_at: started,
                duration,
                trigger: lf.reason,
            })
        });

        SystemStatus {
            version: self.renovate_version,
            boot_time: boot_time.unwrap_or_default(),
            uptime,
            queue_size: self.jobs.queue.size,
            running_jobs: self.jobs.queue.in_progress.len() as u64,
            failed_jobs: 0, // not in this endpoint
            last_finished,
        }
    }
}

impl JobQueueResponse {
    pub fn into_jobs(self) -> Vec<Job> {
        let mut jobs = Vec::new();
        for j in self.running {
            jobs.push(Job {
                id: j.job_id,
                repo: j.repository,
                state: JobState::Running,
                started_at: None,
                duration: None,
                trigger: None,
            });
        }
        for j in self.pending {
            jobs.push(Job {
                id: j.job_id,
                repo: j.repository,
                state: JobState::Pending,
                started_at: None,
                duration: None,
                trigger: None,
            });
        }
        jobs
    }
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

pub struct RenovateClient {
    client: Client,
    base_url: String,
    secret: String,
}

impl RenovateClient {
    pub fn new(base_url: impl Into<String>, secret: impl Into<String>) -> Self {
        let mut url = base_url.into();
        while url.ends_with('/') {
            url.pop();
        }
        Self {
            client: Client::new(),
            base_url: url,
            secret: secret.into(),
        }
    }

    pub async fn get_status(&self) -> Result<SystemStatus> {
        let resp: StatusResponse = self
            .client
            .get(format!("{}/system/v1/status", self.base_url))
            .bearer_auth(&self.secret)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;
        Ok(resp.into_status())
    }

    pub async fn get_jobs(&self) -> Result<Vec<Job>> {
        let resp: JobQueueResponse = self
            .client
            .get(format!("{}/system/v1/jobs/queue", self.base_url))
            .bearer_auth(&self.secret)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;
        Ok(resp.into_jobs())
    }

    pub async fn trigger_sync(&self) -> Result<()> {
        self.client
            .post(format!("{}/system/v1/sync", self.base_url))
            .bearer_auth(&self.secret)
            .send()
            .await?
            .error_for_status()?;
        Ok(())
    }

    pub async fn purge_jobs(&self) -> Result<()> {
        self.client
            .post(format!("{}/system/v1/jobs/purge", self.base_url))
            .bearer_auth(&self.secret)
            .send()
            .await?
            .error_for_status()?;
        Ok(())
    }
}
