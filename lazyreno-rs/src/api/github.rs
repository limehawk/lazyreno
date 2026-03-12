use anyhow::{Context, Result};
use octocrab::params;
use octocrab::Octocrab;
use serde::Deserialize;

use crate::types::{PR, Repo, UpdateType};

/// Minimal combined-status response for commit status checks.
#[derive(Debug, Deserialize)]
struct CombinedStatus {
    state: String,
}

pub struct GithubClient {
    octocrab: Octocrab,
    owner: String,
}

impl GithubClient {
    pub fn new(token: impl Into<String>, owner: impl Into<String>) -> Result<Self> {
        let octocrab = Octocrab::builder()
            .personal_token(token.into())
            .build()
            .context("building octocrab client")?;
        Ok(Self {
            octocrab,
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

    /// List all non-archived repos for the configured owner (org).
    pub async fn list_repos(&self) -> Result<Vec<Repo>> {
        let page = self
            .octocrab
            .orgs(&self.owner)
            .list_repos()
            .per_page(100)
            .send()
            .await
            .context("listing org repos (first page)")?;

        let all = self
            .octocrab
            .all_pages(page)
            .await
            .context("paginating org repos")?;

        let repos: Vec<Repo> = all
            .into_iter()
            .filter(|r| r.archived != Some(true))
            .map(|r| {
                let full_name = r
                    .full_name
                    .clone()
                    .unwrap_or_else(|| format!("{}/{}", self.owner, r.name));
                Repo {
                    name: r.name.clone(),
                    full_name,
                    pr_count: 0, // filled in after PR fetch
                }
            })
            .collect();

        Ok(repos)
    }

    /// List open PRs for a given repo, classifying UpdateType at fetch time.
    pub async fn list_open_prs(&self, repo_name: &str) -> Result<Vec<PR>> {
        let (owner, repo) = self.split_repo(repo_name);

        let page = self
            .octocrab
            .pulls(owner, repo)
            .list()
            .state(params::State::Open)
            .per_page(100)
            .send()
            .await
            .with_context(|| format!("listing PRs for {}/{}", owner, repo))?;

        let all = self
            .octocrab
            .all_pages(page)
            .await
            .with_context(|| format!("paginating PRs for {}/{}", owner, repo))?;

        let full_name = format!("{}/{}", owner, repo);
        let prs: Vec<PR> = all
            .into_iter()
            .map(|pr| {
                let labels: Vec<String> = pr
                    .labels
                    .as_deref()
                    .unwrap_or_default()
                    .iter()
                    .map(|l| l.name.clone())
                    .collect();
                let title = pr.title.clone().unwrap_or_default();
                let update_type = UpdateType::classify(&labels, &title);

                PR {
                    number: pr.number,
                    title,
                    repo: full_name.clone(),
                    branch: pr.head.ref_field.clone(),
                    base: pr.base.ref_field.clone(),
                    url: pr
                        .html_url
                        .as_ref()
                        .map(|u| u.to_string())
                        .unwrap_or_default(),
                    created_at: pr.created_at.unwrap_or_else(chrono::Utc::now),
                    update_type,
                    mergeable: None,  // requires individual PR fetch
                    checks_pass: None, // requires separate status call
                }
            })
            .collect();

        Ok(prs)
    }

    /// Check if all combined commit statuses pass for a given SHA.
    pub async fn get_checks_pass(&self, repo_name: &str, sha: &str) -> Result<bool> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("/repos/{}/{}/commits/{}/status", owner, repo, sha);
        let status: CombinedStatus = self
            .octocrab
            .get(url, None::<&()>)
            .await
            .context("fetching combined status")?;
        Ok(status.state == "success")
    }

    /// Check if a PR is mergeable by fetching its details.
    pub async fn check_mergeable(&self, repo_name: &str, number: u64) -> Result<bool> {
        let (owner, repo) = self.split_repo(repo_name);
        let pr = self
            .octocrab
            .pulls(owner, repo)
            .get(number)
            .await
            .with_context(|| format!("fetching PR #{} in {}/{}", number, owner, repo))?;
        Ok(pr.mergeable.unwrap_or(false))
    }

    /// Merge a PR using the merge method.
    pub async fn merge_pr(&self, repo_name: &str, number: u64) -> Result<()> {
        let (owner, repo) = self.split_repo(repo_name);
        self.octocrab
            .pulls(owner, repo)
            .merge(number)
            .method(params::pulls::MergeMethod::Merge)
            .send()
            .await
            .with_context(|| format!("merging PR #{} in {}/{}", number, owner, repo))?;
        Ok(())
    }

    /// Close a PR by updating its state to Closed.
    pub async fn close_pr(&self, repo_name: &str, number: u64) -> Result<()> {
        let (owner, repo) = self.split_repo(repo_name);
        self.octocrab
            .pulls(owner, repo)
            .update(number)
            .state(params::pulls::State::Closed)
            .send()
            .await
            .with_context(|| format!("closing PR #{} in {}/{}", number, owner, repo))?;
        Ok(())
    }

    /// Delete a branch (best-effort, errors ignored).
    pub async fn delete_branch(&self, repo_name: &str, branch: &str) -> Result<()> {
        let (owner, repo) = self.split_repo(repo_name);
        let url = format!("/repos/{}/{}/git/refs/heads/{}", owner, repo, branch);
        // Best-effort: ignore errors from already-deleted branches.
        let _ = self
            .octocrab
            .delete::<serde_json::Value, _, _>(url, None::<&()>)
            .await;
        Ok(())
    }
}
