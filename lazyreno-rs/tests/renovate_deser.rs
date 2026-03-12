use lazyreno::api::renovate::{JobResponse, StatusResponse};

#[test]
fn deserialize_status_response() {
    let json = include_str!("fixtures/renovate_status.json");
    let resp: StatusResponse = serde_json::from_str(json).unwrap();
    assert_eq!(resp.version, "39.50.0");
    assert_eq!(resp.queue_size, 3);
    assert_eq!(resp.running_jobs, 1);
    assert_eq!(resp.failed_jobs, 0);
    assert!(resp.last_finished_job.is_some());
    let job = resp.last_finished_job.unwrap();
    assert_eq!(job.id, "job-001");
    assert_eq!(job.repository, "org/backend");
    assert_eq!(job.status, "finished");
    assert_eq!(job.duration, Some(45));
}

#[test]
fn deserialize_jobs_response() {
    let json = include_str!("fixtures/renovate_jobs.json");
    let resp: Vec<JobResponse> = serde_json::from_str(json).unwrap();
    assert_eq!(resp.len(), 2);
    assert_eq!(resp[0].status, "running");
    assert_eq!(resp[0].repository, "org/frontend");
    assert!(resp[0].started_at.is_some());
    assert_eq!(resp[1].status, "pending");
    assert!(resp[1].started_at.is_none());
    assert!(resp[1].duration.is_none());
}

#[test]
fn status_response_into_status() {
    let json = include_str!("fixtures/renovate_status.json");
    let resp: StatusResponse = serde_json::from_str(json).unwrap();
    let status = resp.into_status();
    assert_eq!(status.version, "39.50.0");
    assert_eq!(status.queue_size, 3);
    assert!(status.last_finished.is_some());
    let job = status.last_finished.unwrap();
    assert_eq!(job.repo, "org/backend");
    assert_eq!(job.duration, Some(std::time::Duration::from_secs(45)));
}

#[test]
fn job_response_into_job() {
    let json = include_str!("fixtures/renovate_jobs.json");
    let resps: Vec<JobResponse> = serde_json::from_str(json).unwrap();
    let job = resps.into_iter().next().unwrap().into_job();
    assert_eq!(job.repo, "org/frontend");
    assert!(matches!(job.state, lazyreno::types::JobState::Running));
    assert_eq!(job.trigger, Some("webhook".to_string()));
}
