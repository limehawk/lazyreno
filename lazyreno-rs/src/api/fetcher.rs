use std::collections::HashMap;
use std::sync::Arc;

use tokio::sync::{mpsc, Semaphore};
use tokio_util::sync::CancellationToken;
use tracing::{error, info};

use crate::api::github::GithubClient;
use crate::api::renovate::RenovateClient;
use crate::config::Config;
use crate::types::{FetchResult, PR, Repo};

/// Maximum concurrent PR-fetch tasks per cycle.
const MAX_CONCURRENT_PR_FETCHES: usize = 5;

/// Run the background fetch loop. Sends a `FetchResult` on each tick.
/// Exits when the cancellation token fires or the receiver is dropped.
pub async fn run_fetcher(
    config: Arc<Config>,
    github: Arc<GithubClient>,
    renovate: Arc<RenovateClient>,
    tx: mpsc::Sender<FetchResult>,
    cancel: CancellationToken,
) {
    let mut interval = tokio::time::interval(config.ui.refresh_interval);
    loop {
        tokio::select! {
            _ = cancel.cancelled() => {
                info!("fetcher: cancelled, shutting down");
                break;
            }
            _ = interval.tick() => {
                let result = fetch_all(github.clone(), &renovate).await;
                if tx.send(result).await.is_err() {
                    info!("fetcher: receiver dropped, shutting down");
                    break;
                }
            }
        }
    }
}

/// Perform a single full fetch cycle: repos, PRs, status, jobs.
/// Each top-level field is independently failable (partial failures OK).
async fn fetch_all(github: Arc<GithubClient>, renovate: &RenovateClient) -> FetchResult {
    // 1. Fetch repos
    let repos_result = github.list_repos().await;

    // 2. Fetch PRs concurrently (bounded) if repos succeeded
    let prs_result = match &repos_result {
        Ok(repos) => fetch_all_prs(github.clone(), repos).await,
        Err(_) => Err(anyhow::anyhow!("skipped PR fetch: repos failed")),
    };

    // 3. Fetch status and jobs (independent, can run concurrently)
    let (status_result, jobs_result) =
        tokio::join!(renovate.get_status(), renovate.get_jobs());

    FetchResult {
        repos: repos_result,
        prs: prs_result,
        status: status_result,
        jobs: jobs_result,
    }
}

/// Fetch open PRs for all repos with bounded concurrency via a semaphore.
async fn fetch_all_prs(
    github: Arc<GithubClient>,
    repos: &[Repo],
) -> anyhow::Result<HashMap<String, Vec<PR>>> {
    let semaphore = Arc::new(Semaphore::new(MAX_CONCURRENT_PR_FETCHES));
    let mut handles = Vec::with_capacity(repos.len());

    for repo in repos {
        let gh = github.clone();
        let sem = semaphore.clone();
        let full_name = repo.full_name.clone();

        handles.push(tokio::spawn(async move {
            let _permit = sem.acquire().await;
            let result = gh.list_open_prs(&full_name).await;
            (full_name, result)
        }));
    }

    let mut map = HashMap::new();
    let mut errors = Vec::new();

    for handle in handles {
        match handle.await {
            Ok((name, Ok(prs))) => {
                map.insert(name, prs);
            }
            Ok((name, Err(e))) => {
                error!(repo = %name, error = %e, "failed to fetch PRs");
                errors.push(format!("{}: {}", name, e));
            }
            Err(e) => {
                error!(error = %e, "PR fetch task panicked");
                errors.push(format!("task panic: {}", e));
            }
        }
    }

    if errors.is_empty() {
        Ok(map)
    } else {
        // Partial success: include what we got in the map, but report errors.
        // For now, return Ok with whatever we collected — callers can check
        // missing repos. If ALL failed, still return what we have (empty map).
        // We log errors above; return Ok so partial data is usable.
        Ok(map)
    }
}
