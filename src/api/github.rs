use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use reqwest::Client;
use serde::Deserialize;

use crate::types::{PR, Repo, UpdateType};

/// Three-way mergeable status for batch merge operations.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Mergeability {
    /// PR can be merged now.
    Ready,
    /// PR has conflicts — won't resolve without intervention.
    Conflict,
    /// GitHub hasn't computed mergeability yet — may become ready.
    Unknown,
}

/// Minimal combined-status response for commit status checks.
#[derive(Debug, Deserialize)]
#[allow(dead_code)]
struct CombinedStatus {
    state: String,
}

/// Minimal PR response from the GitHub REST API.
#[derive(Debug, Deserialize)]
struct GhPullRequest {
    number: u64,
    title: Option<String>,
    html_url: Option<String>,
    created_at: Option<DateTime<Utc>>,
    mergeable: Option<bool>,
    labels: Option<Vec<GhLabel>>,
    head: GhRef,
    base: GhRef,
}

#[derive(Debug, Deserialize)]
struct GhLabel {
    name: String,
}

#[derive(Debug, Deserialize)]
struct GhRef {
    #[serde(rename = "ref")]
    ref_field: String,
    #[serde(default)]
    sha: String,
}

pub struct GithubClient {
    client: Client,
    owner: String,
}

impl GithubClient {
    pub fn new(token: impl Into<String>, owner: impl Into<String>) -> Result<Self> {
        let client = Client::builder()
            .user_agent("lazyreno")
            .default_headers({
                let mut h = reqwest::header::HeaderMap::new();
                let val = format!("Bearer {}", token.into());
                h.insert(
                    reqwest::header::AUTHORIZATION,
                    val.parse().context("invalid token")?,
                );
                h.insert(
                    reqwest::header::ACCEPT,
                    "application/vnd.github+json"
                        .parse()
                        .expect("static header"),
                );
                h
            })
            .build()
            .context("building HTTP client")?;
        Ok(Self {
            client,
            owner: owner.into(),
        })
    }

    /// Split "owner/repo" into (owner, repo). Falls back to self.owner if
    /// no slash is present.
    fn split_repo<'a>(&'a self, repo_name: &'a str) -> (&'a str, &'a str) {
        match repo_name.split_once('/') {
            Some((owner, repo)) => (owner, repo),
            None => (self.owner.as_str(), repo_name),
        }
    }

    /// GET a paginated list from the GitHub API, following `per_page` + `page`.
    async fn get_paginated<T: serde::de::DeserializeOwned>(
        &self,
        base_url: &str,
        separator: char,
    ) -> Result<Vec<T>> {
        let mut all = Vec::new();
        let mut page = 1u32;
        loop {
            let url = format!("{base_url}{separator}per_page=100&page={page}");
            let items: Vec<T> = self
                .client
                .get(&url)
                .send()
                .await
                .context("GitHub API request")?
                .error_for_status()
                .context("GitHub API error")?
                .json()
                .await
                .context("parsing GitHub response")?;
            if items.is_empty() {
                break;
            }
            all.extend(items);
            page += 1;
        }
        Ok(all)
    }

    /// List all non-archived repos for the configured owner (user or org).
    pub async fn list_repos(&self) -> Result<Vec<Repo>> {
        // Use the authenticated /user/repos endpoint so private repos are
        // included, then filter to the configured owner.
        let url = "https://api.github.com/user/repos".to_string();
        let items: Vec<serde_json::Value> = self
            .get_paginated(&url, '?')
            .await
            .context("listing repos")?;

        // Keep only repos belonging to the configured owner.
        let items: Vec<serde_json::Value> = items
            .into_iter()
            .filter(|r| {
                r.get("owner")
                    .and_then(|o| o.get("login"))
                    .and_then(|l| l.as_str())
                    .map(|login| login.eq_ignore_ascii_case(&self.owner))
                    .unwrap_or(false)
            })
            .collect();

        let repos = items
            .iter()
            .filter(|r| {
                !r.get("archived")
                    .and_then(|v| v.as_bool())
                    .unwrap_or(false)
            })
            .map(|r| {
                let name = r
                    .get("name")
                    .and_then(|v| v.as_str())
                    .unwrap_or_default()
                    .to_string();
                let full_name = r
                    .get("full_name")
                    .and_then(|v| v.as_str())
                    .map(|s| s.to_string())
                    .unwrap_or_else(|| format!("{}/{}", self.owner, name));
                let fork = r
                    .get("fork")
                    .and_then(|v| v.as_bool())
                    .unwrap_or(false);
                Repo {
                    name,
                    full_name,
                    pr_count: 0,
                    fork,
                }
            })
            .collect();

        Ok(repos)
    }

    /// List open PRs for a given repo, classifying UpdateType at fetch time.
    pub async fn list_open_prs(&self, repo_name: &str) -> Result<Vec<PR>> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!(
            "https://api.github.com/repos/{owner}/{repo}/pulls?state=open"
        );
        let items: Vec<GhPullRequest> = self
            .get_paginated(&url, '&')
            .await
            .with_context(|| format!("listing PRs for {owner}/{repo}"))?;

        let full_name = format!("{owner}/{repo}");
        let prs = items
            .into_iter()
            .map(|pr| {
                let labels: Vec<String> = pr
                    .labels
                    .as_deref()
                    .unwrap_or_default()
                    .iter()
                    .map(|l| l.name.clone())
                    .collect();
                let title = pr.title.unwrap_or_default();
                let update_type = UpdateType::classify(&labels, &title);
                PR {
                    number: pr.number,
                    title,
                    repo: full_name.clone(),
                    branch: pr.head.ref_field,
                    base: pr.base.ref_field,
                    head_sha: pr.head.sha,
                    url: pr.html_url.unwrap_or_default(),
                    created_at: pr.created_at.unwrap_or_else(Utc::now),
                    update_type,
                    mergeable: None,
                    checks_pass: None,
                }
            })
            .collect();

        Ok(prs)
    }

    /// Check if all combined commit statuses pass for a given SHA.
    pub async fn get_checks_pass(&self, repo_name: &str, sha: &str) -> Result<bool> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("https://api.github.com/repos/{owner}/{repo}/commits/{sha}/status");
        let status: CombinedStatus = self
            .client
            .get(&url)
            .send()
            .await?
            .error_for_status()
            .context("fetching combined status")?
            .json()
            .await?;
        Ok(status.state == "success")
    }

    /// Check if a PR is mergeable by fetching its details.
    /// GitHub computes mergeability lazily — `mergeable` may be `null` on the
    /// first request. We retry a few times with a short delay.
    pub async fn check_mergeable(&self, repo_name: &str, number: u64) -> Result<bool> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("https://api.github.com/repos/{owner}/{repo}/pulls/{number}");

        for attempt in 0..5 {
            if attempt > 0 {
                tokio::time::sleep(std::time::Duration::from_secs(2)).await;
            }
            let pr: GhPullRequest = self
                .client
                .get(&url)
                .send()
                .await?
                .error_for_status()
                .with_context(|| format!("fetching PR #{number} in {owner}/{repo}"))?
                .json()
                .await?;
            if let Some(mergeable) = pr.mergeable {
                return Ok(mergeable);
            }
        }
        // After retries, treat unknown as not mergeable.
        Ok(false)
    }

    /// Single-shot mergeable check — no retries. Returns three-way result
    /// so batch callers can distinguish "has conflicts" from "not computed yet".
    pub async fn check_mergeable_once(
        &self,
        repo_name: &str,
        number: u64,
    ) -> Result<Mergeability> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("https://api.github.com/repos/{owner}/{repo}/pulls/{number}");
        let pr: GhPullRequest = self
            .client
            .get(&url)
            .send()
            .await?
            .error_for_status()
            .with_context(|| format!("fetching PR #{number} in {owner}/{repo}"))?
            .json()
            .await?;
        Ok(match pr.mergeable {
            Some(true) => Mergeability::Ready,
            Some(false) => Mergeability::Conflict,
            None => Mergeability::Unknown,
        })
    }

    /// Merge a PR using the merge method.
    pub async fn merge_pr(&self, repo_name: &str, number: u64) -> Result<()> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("https://api.github.com/repos/{owner}/{repo}/pulls/{number}/merge");
        let resp = self
            .client
            .put(&url)
            .json(&serde_json::json!({ "merge_method": "merge" }))
            .send()
            .await
            .with_context(|| format!("merging PR #{number} in {owner}/{repo}"))?;

        if !resp.status().is_success() {
            let status = resp.status();
            let body: serde_json::Value = resp.json().await.unwrap_or_default();
            let msg = body["message"].as_str().unwrap_or("unknown error");
            anyhow::bail!("PR #{number} in {owner}/{repo}: {status} — {msg}");
        }
        Ok(())
    }

    /// Close a PR by updating its state to Closed.
    pub async fn close_pr(&self, repo_name: &str, number: u64) -> Result<()> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("https://api.github.com/repos/{owner}/{repo}/pulls/{number}");
        self.client
            .patch(&url)
            .json(&serde_json::json!({ "state": "closed" }))
            .send()
            .await?
            .error_for_status()
            .with_context(|| format!("closing PR #{number} in {owner}/{repo}"))?;
        Ok(())
    }

    /// Post a comment on a PR (used for Renovate commands like /rebase).
    pub async fn post_comment(&self, repo_name: &str, number: u64, body: &str) -> Result<()> {
        let (owner, repo) = self.split_repo(repo_name);
        let url =
            format!("https://api.github.com/repos/{owner}/{repo}/issues/{number}/comments");
        self.client
            .post(&url)
            .json(&serde_json::json!({ "body": body }))
            .send()
            .await?
            .error_for_status()
            .with_context(|| format!("commenting on PR #{number} in {owner}/{repo}"))?;
        Ok(())
    }

    /// Delete a branch (best-effort, errors ignored).
    pub async fn delete_branch(&self, repo_name: &str, branch: &str) -> Result<()> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("https://api.github.com/repos/{owner}/{repo}/git/refs/heads/{branch}");
        let _ = self.client.delete(&url).send().await;
        Ok(())
    }
}
