use std::collections::HashMap;
use std::sync::Arc;

use tokio::sync::{Semaphore, mpsc};
use tokio_util::sync::CancellationToken;
use tracing::{error, info, warn};

use crate::api::github::GithubClient;
use crate::api::renovate::RenovateClient;
use crate::config::Config;
use crate::types::{FetchResult, PR, Repo};

/// Maximum concurrent PR-fetch tasks per cycle.
const MAX_CONCURRENT_PR_FETCHES: usize = 5;

/// Maximum concurrent enrichment tasks (mergeable + checks) per cycle.
const MAX_CONCURRENT_ENRICHMENTS: usize = 5;

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
    let (status_result, jobs_result) = tokio::join!(renovate.get_status(), renovate.get_jobs());

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

    // Enrich PRs with mergeable + checks status (best-effort, bounded concurrency).
    enrich_prs(&github, &mut map).await;

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

/// Enrich PRs with mergeable status and checks pass (one-shot, no retries).
/// Failures are silently ignored — fields stay `None`.
async fn enrich_prs(github: &Arc<GithubClient>, prs_by_repo: &mut HashMap<String, Vec<PR>>) {
    let semaphore = Arc::new(Semaphore::new(MAX_CONCURRENT_ENRICHMENTS));

    // Collect all (repo, number, sha) tuples for enrichment.
    let targets: Vec<(String, u64, String)> = prs_by_repo
        .values()
        .flat_map(|prs| {
            prs.iter()
                .map(|pr| (pr.repo.clone(), pr.number, pr.head_sha.clone()))
        })
        .collect();

    // Spawn enrichment tasks.
    let mut handles = Vec::with_capacity(targets.len());
    for (repo, number, sha) in &targets {
        let gh = github.clone();
        let sem = semaphore.clone();
        let repo = repo.clone();
        let number = *number;
        let sha = sha.clone();

        handles.push(tokio::spawn(async move {
            let _permit = sem.acquire().await;
            let mergeable = gh.check_mergeable_once(&repo, number).await.ok();
            let checks = if sha.is_empty() {
                None
            } else {
                gh.get_checks_pass(&repo, &sha).await.ok()
            };
            (repo, number, mergeable, checks)
        }));
    }

    // Collect results and update PRs in place.
    for handle in handles {
        match handle.await {
            Ok((repo, number, mergeable, checks)) => {
                if let Some(prs) = prs_by_repo.get_mut(&repo) {
                    if let Some(pr) = prs.iter_mut().find(|p| p.number == number) {
                        if let Some(m) = mergeable {
                            pr.mergeable = match m {
                                crate::api::github::Mergeability::Ready => Some(true),
                                crate::api::github::Mergeability::Conflict => Some(false),
                                crate::api::github::Mergeability::Unknown => None,
                            };
                        }
                        pr.checks_pass = checks;
                    }
                }
            }
            Err(e) => {
                warn!(error = %e, "enrichment task panicked");
            }
        }
    }
}
