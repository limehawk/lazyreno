use lazyreno::api::renovate::{JobQueueResponse, StatusResponse};

#[test]
fn deserialize_status_response() {
    let json = include_str!("fixtures/renovate_status.json");
    let resp: StatusResponse = serde_json::from_str(json).unwrap();
    assert_eq!(resp.renovate_version, "39.50.0");
    assert_eq!(resp.jobs.queue.size, 3);
    assert_eq!(resp.jobs.queue.in_progress.len(), 1);
    assert_eq!(resp.jobs.queue.in_progress[0].repository, "org/frontend");
    assert!(resp.jobs.history.last_finished.is_some());
    let lf = resp.jobs.history.last_finished.unwrap();
    assert_eq!(lf.job_id, "job-001");
    assert_eq!(lf.repository, "org/backend");
    assert_eq!(lf.status, "finished");
}

#[test]
fn deserialize_jobs_queue_response() {
    let json = include_str!("fixtures/renovate_jobs.json");
    let resp: JobQueueResponse = serde_json::from_str(json).unwrap();
    assert_eq!(resp.running.len(), 1);
    assert_eq!(resp.running[0].repository, "org/frontend");
    assert_eq!(resp.pending.len(), 1);
    assert_eq!(resp.pending[0].repository, "org/infra");
}

#[test]
fn status_response_into_status() {
    let json = include_str!("fixtures/renovate_status.json");
    let resp: StatusResponse = serde_json::from_str(json).unwrap();
    let status = resp.into_status();
    assert_eq!(status.version, "39.50.0");
    assert_eq!(status.queue_size, 3);
    assert_eq!(status.running_jobs, 1);
    assert!(status.last_finished.is_some());
    let job = status.last_finished.unwrap();
    assert_eq!(job.repo, "org/backend");
    assert_eq!(job.duration, Some(std::time::Duration::from_secs(45)));
}

#[test]
fn jobs_queue_into_jobs() {
    let json = include_str!("fixtures/renovate_jobs.json");
    let resp: JobQueueResponse = serde_json::from_str(json).unwrap();
    let jobs = resp.into_jobs();
    assert_eq!(jobs.len(), 2);
    assert_eq!(jobs[0].repo, "org/frontend");
    assert!(matches!(jobs[0].state, lazyreno::types::JobState::Running));
    assert_eq!(jobs[1].repo, "org/infra");
    assert!(matches!(jobs[1].state, lazyreno::types::JobState::Pending));
}
