use std::collections::HashMap;
use std::sync::Arc;

use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing::error;

use crate::api::github::{GithubClient, Mergeability};
use crate::api::renovate::RenovateClient;
use crate::types::{
    ActionResult, ConfirmAction, FetchResult, FlashMessage, Job, PR, Panel, Repo, SystemStatus,
    UpdateType,
};

/// Root application state.
pub struct App {
    pub repos: Vec<Repo>,
    pub prs: HashMap<String, Vec<PR>>,
    pub selected_repo: usize,
    pub selected_pr: usize,
    pub focused_panel: Panel,
    pub system_status: Option<SystemStatus>,
    pub jobs: Vec<Job>,
    pub flash: Option<FlashMessage>,
    pub activity_log: Vec<FlashMessage>,
    pub confirming: Option<ConfirmAction>,
    pub show_help: bool,
    pub show_all_repos: bool,
    pub all_repos: Vec<Repo>,
    pub all_repos_selected: usize,
    pub all_repos_filter: String,
    #[allow(dead_code)]
    pub filter_text: String,
    pub hide_forks: bool,
    pub running: bool,
    pub loaded: bool,
    pub cancel_token: CancellationToken,
    pub action_tx: mpsc::Sender<ActionResult>,
    pub github: Arc<GithubClient>,
    pub renovate: Arc<RenovateClient>,
}

impl App {
    pub fn new(
        cancel_token: CancellationToken,
        action_tx: mpsc::Sender<ActionResult>,
        github: Arc<GithubClient>,
        renovate: Arc<RenovateClient>,
    ) -> Self {
        Self {
            repos: Vec::new(),
            prs: HashMap::new(),
            selected_repo: 0,
            selected_pr: 0,
            focused_panel: Panel::Sidebar,
            system_status: None,
            jobs: Vec::new(),
            flash: None,
            activity_log: Vec::new(),
            confirming: None,
            show_help: false,
            show_all_repos: false,
            all_repos: Vec::new(),
            all_repos_selected: 0,
            all_repos_filter: String::new(),
            filter_text: String::new(),
            hide_forks: true,
            running: true,
            loaded: false,
            cancel_token,
            action_tx,
            github,
            renovate,
        }
    }

    // -----------------------------------------------------------------------
    // Selection helpers
    // -----------------------------------------------------------------------

    /// Full name of the currently selected repo, if any.
    pub fn selected_repo_name(&self) -> Option<&str> {
        self.repos
            .get(self.selected_repo)
            .map(|r| r.full_name.as_str())
    }

    /// PRs for the currently selected repo (empty slice if none).
    pub fn current_prs(&self) -> &[PR] {
        self.selected_repo_name()
            .and_then(|name| self.prs.get(name))
            .map(|v| v.as_slice())
            .unwrap_or(&[])
    }

    /// The currently highlighted PR, if any.
    pub fn selected_pr(&self) -> Option<&PR> {
        self.current_prs().get(self.selected_pr)
    }

    /// All repos with fork filter applied.
    pub fn visible_all_repos(&self) -> Vec<&Repo> {
        self.all_repos
            .iter()
            .filter(|r| !self.hide_forks || !r.fork)
            .collect()
    }

    // -----------------------------------------------------------------------
    // State update from fetch results
    // -----------------------------------------------------------------------

    /// Apply a complete fetch result to the app state.
    pub fn apply_fetch(&mut self, result: FetchResult) {
        // Repos — store all repos for the overlay, filter to PR-bearing for sidebar.
        match result.repos {
            Ok(repos) => {
                self.all_repos = repos;
            }
            Err(e) => {
                error!(error = %e, "fetch repos failed");
                self.flash = Some(FlashMessage::error(format!("Repo fetch: {e}")));
            }
        }

        // PRs
        match result.prs {
            Ok(prs) => {
                self.prs = prs;
            }
            Err(e) => {
                error!(error = %e, "fetch PRs failed");
                self.flash = Some(FlashMessage::error(format!("PR fetch: {e}")));
            }
        }

        // Rebuild sidebar repos: only those with open PRs, sorted alphabetically.
        let mut sidebar_repos: Vec<Repo> = self
            .all_repos
            .iter()
            .filter(|r| !self.hide_forks || !r.fork)
            .filter(|r| {
                self.prs
                    .get(&r.full_name)
                    .is_some_and(|prs| !prs.is_empty())
            })
            .cloned()
            .map(|mut r| {
                r.pr_count = self.prs.get(&r.full_name).map(|v| v.len()).unwrap_or(0);
                r
            })
            .collect();
        sidebar_repos.sort_by(|a, b| a.full_name.to_lowercase().cmp(&b.full_name.to_lowercase()));
        self.repos = sidebar_repos;

        // Clamp selections.
        self.clamp_selections();

        // System status
        match result.status {
            Ok(status) => self.system_status = Some(status),
            Err(e) => {
                error!(error = %e, "fetch status failed");
                self.flash = Some(FlashMessage::error(format!("Status fetch: {e}")));
            }
        }

        // Jobs
        match result.jobs {
            Ok(jobs) => self.jobs = jobs,
            Err(e) => {
                error!(error = %e, "fetch jobs failed");
                self.flash = Some(FlashMessage::error(format!("Jobs fetch: {e}")));
            }
        }

        self.loaded = true;
    }

    // -----------------------------------------------------------------------
    // State update from action results
    // -----------------------------------------------------------------------

    /// Set flash and append to activity log.
    fn log_flash(&mut self, flash: FlashMessage) {
        self.activity_log.push(flash.clone());
        self.flash = Some(flash);
    }

    /// Apply a completed user action to app state.
    pub fn apply_action(&mut self, result: ActionResult) {
        match result {
            ActionResult::PrMerged { repo, number } => {
                self.remove_pr(&repo, number);
                self.log_flash(FlashMessage::success(format!(
                    "Merged PR #{number} in {repo}"
                )));
            }
            ActionResult::PrClosed { repo, number } => {
                self.remove_pr(&repo, number);
                self.log_flash(FlashMessage::success(format!(
                    "Closed PR #{number} in {repo}"
                )));
            }
            ActionResult::PrCommented { repo, number, command } => {
                self.log_flash(FlashMessage::success(format!(
                    "Sent {command} to PR #{number} in {repo}"
                )));
            }
            ActionResult::AllSafeMerged { repo, count, skipped } => {
                let mut msg = format!("Merged {count} safe PRs in {repo}");
                if skipped > 0 {
                    msg.push_str(&format!(", {skipped} not mergeable"));
                }
                self.log_flash(FlashMessage::success(msg));
            }
            ActionResult::AllMerged { repo, count, skipped } => {
                let mut msg = format!("Merged {count} PRs in {repo}");
                if skipped > 0 {
                    msg.push_str(&format!(", {skipped} not mergeable"));
                }
                self.log_flash(FlashMessage::success(msg));
            }
            ActionResult::AllRebased { repo, count, failed } => {
                let mut msg = format!("Rebased {count} PRs in {repo}");
                if failed > 0 {
                    msg.push_str(&format!(", {failed} failed"));
                }
                self.log_flash(FlashMessage::success(msg));
            }
            ActionResult::SyncTriggered => {
                self.log_flash(FlashMessage::success("Renovate sync triggered"));
            }
            ActionResult::JobsPurged => {
                self.log_flash(FlashMessage::success("Finished jobs purged"));
            }
            ActionResult::Error(msg) => {
                self.log_flash(FlashMessage::error(msg));
            }
        }
    }

    /// Remove a PR from local state and clean up empty repos.
    fn remove_pr(&mut self, repo: &str, number: u64) {
        if let Some(prs) = self.prs.get_mut(repo) {
            prs.retain(|pr| pr.number != number);
            if prs.is_empty() {
                self.repos.retain(|r| r.full_name != repo);
            } else if let Some(r) = self.repos.iter_mut().find(|r| r.full_name == repo) {
                r.pr_count = prs.len();
            }
        }
        self.clamp_selections();
    }

    // -----------------------------------------------------------------------
    // Navigation
    // -----------------------------------------------------------------------

    pub fn move_selection_down(&mut self) {
        match self.focused_panel {
            Panel::Sidebar => {
                if !self.repos.is_empty() && self.selected_repo < self.repos.len() - 1 {
                    self.selected_repo += 1;
                    self.selected_pr = 0;
                }
            }
            Panel::PrTable => {
                let len = self.current_prs().len();
                if len > 0 && self.selected_pr < len - 1 {
                    self.selected_pr += 1;
                }
            }
            Panel::Detail => {}
        }
    }

    pub fn move_selection_up(&mut self) {
        match self.focused_panel {
            Panel::Sidebar => {
                if self.selected_repo > 0 {
                    self.selected_repo -= 1;
                    self.selected_pr = 0;
                }
            }
            Panel::PrTable => {
                if self.selected_pr > 0 {
                    self.selected_pr -= 1;
                }
            }
            Panel::Detail => {}
        }
    }

    pub fn jump_top(&mut self) {
        match self.focused_panel {
            Panel::Sidebar => {
                self.selected_repo = 0;
                self.selected_pr = 0;
            }
            Panel::PrTable => {
                self.selected_pr = 0;
            }
            Panel::Detail => {}
        }
    }

    pub fn jump_bottom(&mut self) {
        match self.focused_panel {
            Panel::Sidebar => {
                if !self.repos.is_empty() {
                    self.selected_repo = self.repos.len() - 1;
                    self.selected_pr = 0;
                }
            }
            Panel::PrTable => {
                let len = self.current_prs().len();
                if len > 0 {
                    self.selected_pr = len - 1;
                }
            }
            Panel::Detail => {}
        }
    }

    pub fn half_page_down(&mut self, visible_rows: usize) {
        let half = visible_rows / 2;
        match self.focused_panel {
            Panel::Sidebar => {
                if !self.repos.is_empty() {
                    self.selected_repo = (self.selected_repo + half).min(self.repos.len() - 1);
                    self.selected_pr = 0;
                }
            }
            Panel::PrTable => {
                let len = self.current_prs().len();
                if len > 0 {
                    self.selected_pr = (self.selected_pr + half).min(len - 1);
                }
            }
            Panel::Detail => {}
        }
    }

    pub fn half_page_up(&mut self, visible_rows: usize) {
        let half = visible_rows / 2;
        match self.focused_panel {
            Panel::Sidebar => {
                self.selected_repo = self.selected_repo.saturating_sub(half);
                self.selected_pr = 0;
            }
            Panel::PrTable => {
                self.selected_pr = self.selected_pr.saturating_sub(half);
            }
            Panel::Detail => {}
        }
    }

    /// Rebuild sidebar repos from the full repo list (e.g. after toggling fork filter).
    pub fn rebuild_sidebar(&mut self) {
        let mut sidebar_repos: Vec<Repo> = self
            .all_repos
            .iter()
            .filter(|r| !self.hide_forks || !r.fork)
            .filter(|r| {
                self.prs
                    .get(&r.full_name)
                    .is_some_and(|prs| !prs.is_empty())
            })
            .cloned()
            .map(|mut r| {
                r.pr_count = self.prs.get(&r.full_name).map(|v| v.len()).unwrap_or(0);
                r
            })
            .collect();
        sidebar_repos.sort_by(|a, b| a.full_name.to_lowercase().cmp(&b.full_name.to_lowercase()));
        self.repos = sidebar_repos;
        self.clamp_selections();
    }

    /// Clear the flash message if it has expired.
    pub fn clear_expired_flash(&mut self) {
        if self.flash.as_ref().is_some_and(|f| f.is_expired()) {
            self.flash = None;
        }
    }

    // -----------------------------------------------------------------------
    // Async action dispatch
    // -----------------------------------------------------------------------

    /// Merge a single PR (check mergeable first).
    pub fn dispatch_merge(&self, number: u64, repo: String) {
        let gh = self.github.clone();
        let tx = self.action_tx.clone();
        tokio::spawn(async move {
            let result = async {
                let mergeable = gh.check_mergeable(&repo, number).await?;
                if !mergeable {
                    anyhow::bail!("PR #{number} in {repo} is not mergeable");
                }
                gh.merge_pr(&repo, number).await?;
                Ok(())
            }
            .await;

            let action = match result {
                Ok(()) => ActionResult::PrMerged { repo, number },
                Err(e) => ActionResult::Error(format!("Merge PR #{number}: {e}")),
            };
            let _ = tx.send(action).await;
        });
    }

    /// Close a PR and delete its branch (best-effort).
    pub fn dispatch_close(&self, number: u64, repo: String, branch: String) {
        let gh = self.github.clone();
        let tx = self.action_tx.clone();
        tokio::spawn(async move {
            let result: anyhow::Result<()> = async {
                gh.close_pr(&repo, number).await?;
                // Best-effort branch deletion.
                let _ = gh.delete_branch(&repo, &branch).await;
                Ok(())
            }
            .await;

            let action = match result {
                Ok(()) => ActionResult::PrClosed { repo, number },
                Err(e) => ActionResult::Error(format!("Close PR #{number}: {e}")),
            };
            let _ = tx.send(action).await;
        });
    }

    /// Post a Renovate command comment on a PR (e.g. /rebase, /recreate, /retry).
    pub fn dispatch_renovate_command(&self, number: u64, repo: String, command: String) {
        let gh = self.github.clone();
        let tx = self.action_tx.clone();
        tokio::spawn(async move {
            let result = gh.post_comment(&repo, number, &command).await;
            let action = match result {
                Ok(()) => ActionResult::PrCommented {
                    repo,
                    number,
                    command,
                },
                Err(e) => ActionResult::Error(format!("{command} PR #{number}: {e}")),
            };
            let _ = tx.send(action).await;
        });
    }

    /// Post /rebase on all PRs for a given repo.
    pub fn dispatch_rebase_all(&self, repo: String) {
        let all_prs: Vec<(u64, String)> = self
            .prs
            .get(&repo)
            .map(|prs| prs.iter().map(|pr| (pr.number, pr.repo.clone())).collect())
            .unwrap_or_default();

        let gh = self.github.clone();
        let tx = self.action_tx.clone();
        let repo_name = repo.clone();
        tokio::spawn(async move {
            let mut count = 0usize;
            let mut failed = 0usize;

            for (number, repo) in &all_prs {
                match gh.post_comment(repo, *number, "/rebase").await {
                    Ok(()) => count += 1,
                    Err(_) => failed += 1,
                }
            }

            let _ = tx
                .send(ActionResult::AllRebased {
                    repo: repo_name,
                    count,
                    failed,
                })
                .await;
        });
    }

    /// Merge all safe PRs for a given repo.
    pub fn dispatch_merge_all_safe(&self, repo: String) {
        let safe_prs: Vec<(u64, String)> = self
            .prs
            .get(&repo)
            .map(|prs| {
                prs.iter()
                    .filter(|pr| pr.is_safe())
                    .map(|pr| (pr.number, pr.repo.clone()))
                    .collect()
            })
            .unwrap_or_default();

        let gh = self.github.clone();
        let tx = self.action_tx.clone();
        let repo_name = repo.clone();
        tokio::spawn(async move {
            let mut merged = 0usize;
            let mut skipped = 0usize;
            let mut errors = Vec::new();
            let mut remaining = safe_prs;

            // Queue-based merge: check once per PR, re-queue unknowns.
            // Processing other PRs provides the natural delay GitHub needs.
            for _pass in 0..10 {
                let mut unknown = Vec::new();
                let mut made_progress = false;

                for (number, repo) in &remaining {
                    match gh.check_mergeable_once(repo, *number).await {
                        Ok(Mergeability::Ready) => match gh.merge_pr(repo, *number).await {
                            Ok(()) => {
                                merged += 1;
                                made_progress = true;
                                let _ = tx.send(ActionResult::PrMerged {
                                    repo: repo.clone(),
                                    number: *number,
                                }).await;
                            }
                            Err(e) => errors.push(format!("#{number}: {e}")),
                        },
                        Ok(Mergeability::Conflict) => skipped += 1,
                        Ok(Mergeability::Unknown) => unknown.push((*number, repo.clone())),
                        Err(e) => errors.push(format!("#{number}: {e}")),
                    }
                }

                remaining = unknown;
                if remaining.is_empty() {
                    break;
                }
                if !made_progress {
                    // No merges this pass — GitHub may still be computing.
                    // Brief wait before retrying unknowns.
                    tokio::time::sleep(std::time::Duration::from_secs(3)).await;
                }
            }

            skipped += remaining.len();
            let action = if errors.is_empty() {
                ActionResult::AllSafeMerged {
                    repo: repo_name,
                    count: merged,
                    skipped,
                }
            } else {
                ActionResult::Error(format!(
                    "Merged {merged}, skipped {skipped}, failed {}: {}",
                    errors.len(),
                    errors.join("; ")
                ))
            };
            let _ = tx.send(action).await;
        });
    }

    /// Merge all PRs for a given repo (regardless of update type).
    /// Order: patch → minor → major/digest/pin/unknown, then by PR number within each group.
    pub fn dispatch_merge_all(&self, repo: String) {
        let mut all_prs: Vec<(u64, String, UpdateType)> = self
            .prs
            .get(&repo)
            .map(|prs| {
                prs.iter()
                    .map(|pr| (pr.number, pr.repo.clone(), pr.update_type))
                    .collect()
            })
            .unwrap_or_default();
        all_prs.sort_by_key(|(number, _, ut)| (ut.merge_order(), *number));
        let all_prs: Vec<(u64, String)> = all_prs
            .into_iter()
            .map(|(number, repo, _)| (number, repo))
            .collect();

        let gh = self.github.clone();
        let tx = self.action_tx.clone();
        let repo_name = repo.clone();
        tokio::spawn(async move {
            let mut merged = 0usize;
            let mut skipped = 0usize;
            let mut errors = Vec::new();
            let mut remaining = all_prs;

            // Queue-based merge: check once per PR, re-queue unknowns.
            // Processing other PRs provides the natural delay GitHub needs.
            for _pass in 0..10 {
                let mut unknown = Vec::new();
                let mut made_progress = false;

                for (number, repo) in &remaining {
                    match gh.check_mergeable_once(repo, *number).await {
                        Ok(Mergeability::Ready) => match gh.merge_pr(repo, *number).await {
                            Ok(()) => {
                                merged += 1;
                                made_progress = true;
                                let _ = tx.send(ActionResult::PrMerged {
                                    repo: repo.clone(),
                                    number: *number,
                                }).await;
                            }
                            Err(e) => errors.push(format!("#{number}: {e}")),
                        },
                        Ok(Mergeability::Conflict) => skipped += 1,
                        Ok(Mergeability::Unknown) => unknown.push((*number, repo.clone())),
                        Err(e) => errors.push(format!("#{number}: {e}")),
                    }
                }

                remaining = unknown;
                if remaining.is_empty() {
                    break;
                }
                if !made_progress {
                    // No merges this pass — GitHub may still be computing.
                    // Brief wait before retrying unknowns.
                    tokio::time::sleep(std::time::Duration::from_secs(3)).await;
                }
            }

            skipped += remaining.len();
            let action = if errors.is_empty() {
                ActionResult::AllMerged {
                    repo: repo_name.clone(),
                    count: merged,
                    skipped,
                }
            } else {
                ActionResult::Error(format!(
                    "Merged {merged}, skipped {skipped}, failed {}: {}",
                    errors.len(),
                    errors.join("; ")
                ))
            };
            let _ = tx.send(action).await;
        });
    }

    /// Trigger a Renovate sync.
    pub fn dispatch_sync(&self) {
        let ren = self.renovate.clone();
        let tx = self.action_tx.clone();
        tokio::spawn(async move {
            let action = match ren.trigger_sync().await {
                Ok(()) => ActionResult::SyncTriggered,
                Err(e) => ActionResult::Error(format!("Sync: {e}")),
            };
            let _ = tx.send(action).await;
        });
    }

    /// Purge finished/failed jobs.
    pub fn dispatch_purge(&self) {
        let ren = self.renovate.clone();
        let tx = self.action_tx.clone();
        tokio::spawn(async move {
            let action = match ren.purge_jobs().await {
                Ok(()) => ActionResult::JobsPurged,
                Err(e) => ActionResult::Error(format!("Purge: {e}")),
            };
            let _ = tx.send(action).await;
        });
    }

    // -----------------------------------------------------------------------
    // Internal helpers
    // -----------------------------------------------------------------------

    fn clamp_selections(&mut self) {
        if self.repos.is_empty() {
            self.selected_repo = 0;
        } else if self.selected_repo >= self.repos.len() {
            self.selected_repo = self.repos.len() - 1;
        }

        let pr_len = self.current_prs().len();
        if pr_len == 0 {
            self.selected_pr = 0;
        } else if self.selected_pr >= pr_len {
            self.selected_pr = pr_len - 1;
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::UpdateType;
    use chrono::Utc;

    fn make_test_app() -> App {
        let cancel = CancellationToken::new();
        let (action_tx, _action_rx) = mpsc::channel(16);
        let github = Arc::new(GithubClient::new("fake", "fake-org").unwrap());
        let renovate = Arc::new(RenovateClient::new("http://localhost", "secret"));
        App::new(cancel, action_tx, github, renovate)
    }

    fn make_pr(number: u64, repo: &str) -> PR {
        PR {
            number,
            title: format!("PR #{number}"),
            repo: repo.into(),
            branch: "renovate/test".into(),
            base: "main".into(),
            url: format!("https://github.com/{repo}/pull/{number}"),
            created_at: Utc::now(),
            update_type: UpdateType::Minor,
            mergeable: Some(true),
            checks_pass: Some(true),
        }
    }

    #[tokio::test]
    async fn selected_repo_name_empty() {
        let app = make_test_app();
        assert!(app.selected_repo_name().is_none());
    }

    #[tokio::test]
    async fn selected_repo_name_with_repos() {
        let mut app = make_test_app();
        app.repos.push(Repo {
            full_name: "org/alpha".into(),
            name: "alpha".into(),
            pr_count: 1,
            fork: false,
        });
        assert_eq!(app.selected_repo_name(), Some("org/alpha"));
    }

    #[tokio::test]
    async fn current_prs_returns_matching() {
        let mut app = make_test_app();
        app.repos.push(Repo {
            full_name: "org/repo".into(),
            name: "repo".into(),
            pr_count: 2,
            fork: false,
        });
        app.prs.insert(
            "org/repo".into(),
            vec![make_pr(1, "org/repo"), make_pr(2, "org/repo")],
        );
        assert_eq!(app.current_prs().len(), 2);
    }

    #[tokio::test]
    async fn navigation_down_up() {
        let mut app = make_test_app();
        app.repos = vec![
            Repo {
                full_name: "org/a".into(),
                name: "a".into(),
                pr_count: 1,
                fork: false,
            },
            Repo {
                full_name: "org/b".into(),
                name: "b".into(),
                pr_count: 1,
                fork: false,
            },
            Repo {
                full_name: "org/c".into(),
                name: "c".into(),
                pr_count: 1,
                fork: false,
            },
        ];
        assert_eq!(app.selected_repo, 0);

        app.move_selection_down();
        assert_eq!(app.selected_repo, 1);

        app.move_selection_down();
        assert_eq!(app.selected_repo, 2);

        // Should not go past the end.
        app.move_selection_down();
        assert_eq!(app.selected_repo, 2);

        app.move_selection_up();
        assert_eq!(app.selected_repo, 1);

        app.jump_top();
        assert_eq!(app.selected_repo, 0);

        app.jump_bottom();
        assert_eq!(app.selected_repo, 2);
    }

    #[tokio::test]
    async fn apply_action_pr_merged_removes_pr() {
        let mut app = make_test_app();
        app.repos.push(Repo {
            full_name: "org/repo".into(),
            name: "repo".into(),
            pr_count: 2,
            fork: false,
        });
        app.prs.insert(
            "org/repo".into(),
            vec![make_pr(1, "org/repo"), make_pr(2, "org/repo")],
        );

        app.apply_action(ActionResult::PrMerged {
            repo: "org/repo".into(),
            number: 1,
        });

        assert_eq!(app.prs["org/repo"].len(), 1);
        assert_eq!(app.prs["org/repo"][0].number, 2);
        assert!(app.flash.is_some());
    }

    #[tokio::test]
    async fn apply_action_removes_empty_repo_from_sidebar() {
        let mut app = make_test_app();
        app.repos.push(Repo {
            full_name: "org/repo".into(),
            name: "repo".into(),
            pr_count: 1,
            fork: false,
        });
        app.prs
            .insert("org/repo".into(), vec![make_pr(1, "org/repo")]);

        app.apply_action(ActionResult::PrClosed {
            repo: "org/repo".into(),
            number: 1,
        });

        assert!(app.repos.is_empty());
    }

    #[tokio::test]
    async fn half_page_clamps_to_bounds() {
        let mut app = make_test_app();
        app.repos = vec![
            Repo {
                full_name: "org/a".into(),
                name: "a".into(),
                pr_count: 1,
                fork: false,
            },
            Repo {
                full_name: "org/b".into(),
                name: "b".into(),
                pr_count: 1,
                fork: false,
            },
            Repo {
                full_name: "org/c".into(),
                name: "c".into(),
                pr_count: 1,
                fork: false,
            },
        ];

        app.half_page_down(20); // half = 10, should clamp to 2
        assert_eq!(app.selected_repo, 2);

        app.half_page_up(20); // half = 10, should clamp to 0
        assert_eq!(app.selected_repo, 0);
    }
}
