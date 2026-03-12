use std::time::Duration;

use anyhow::Result;
use chrono::{DateTime, Utc};
use reqwest::Client;
use serde::Deserialize;

use crate::types::{Job, JobState, SystemStatus};

// ---------------------------------------------------------------------------
// Response types (camelCase deserialization)
// ---------------------------------------------------------------------------

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StatusResponse {
    pub version: String,
    pub boot_time: DateTime<Utc>,
    pub uptime: String,
    pub queue_size: u64,
    pub running_jobs: u64,
    pub failed_jobs: u64,
    pub last_finished_job: Option<JobResponse>,
}

#[derive(Debug, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct JobResponse {
    pub id: String,
    pub repository: String,
    pub status: String,
    pub started_at: Option<DateTime<Utc>>,
    pub duration: Option<u64>,
    pub trigger: Option<String>,
}

// ---------------------------------------------------------------------------
// Conversion methods
// ---------------------------------------------------------------------------

impl JobResponse {
    pub fn into_job(self) -> Job {
        let state = match self.status.as_str() {
            "running" => JobState::Running,
            "pending" => JobState::Pending,
            "finished" => JobState::Finished,
            "failed" => JobState::Failed,
            _ => JobState::Pending,
        };

        Job {
            id: self.id,
            repo: self.repository,
            state,
            started_at: self.started_at,
            duration: self.duration.map(Duration::from_secs),
            trigger: self.trigger,
        }
    }
}

impl StatusResponse {
    pub fn into_status(self) -> SystemStatus {
        SystemStatus {
            version: self.version,
            boot_time: self.boot_time,
            uptime: self.uptime,
            queue_size: self.queue_size,
            running_jobs: self.running_jobs,
            failed_jobs: self.failed_jobs,
            last_finished: self.last_finished_job.map(|j| j.into_job()),
        }
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
        let resp: Vec<JobResponse> = self
            .client
            .get(format!("{}/system/v1/jobs/queue", self.base_url))
            .bearer_auth(&self.secret)
            .send()
            .await?
            .error_for_status()?
            .json()
            .await?;
        Ok(resp.into_iter().map(JobResponse::into_job).collect())
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
