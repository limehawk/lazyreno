use std::collections::HashMap;
use std::fmt;
use std::time::{Duration, Instant};

use chrono::{DateTime, Utc};

#[derive(Debug, Clone)]
pub struct Repo {
    pub full_name: String,
    pub name: String,
    pub pr_count: usize,
}

#[derive(Debug, Clone)]
pub struct PR {
    pub number: u64,
    pub title: String,
    pub repo: String,
    pub branch: String,
    pub base: String,
    pub url: String,
    pub created_at: DateTime<Utc>,
    pub update_type: UpdateType,
    pub mergeable: Option<bool>,
    pub checks_pass: Option<bool>,
}

impl PR {
    /// A PR is safe to auto-merge if it is Minor or Patch,
    /// confirmed mergeable, and all checks pass.
    pub fn is_safe(&self) -> bool {
        matches!(self.update_type, UpdateType::Minor | UpdateType::Patch)
            && self.mergeable == Some(true)
            && self.checks_pass == Some(true)
    }

    /// Time elapsed since PR was created.
    pub fn age(&self) -> chrono::Duration {
        Utc::now() - self.created_at
    }

    /// Human-friendly age string: "3d", "5h", "12m".
    pub fn age_display(&self) -> String {
        let dur = self.age();
        let days = dur.num_days();
        if days > 0 {
            return format!("{}d", days);
        }
        let hours = dur.num_hours();
        if hours > 0 {
            return format!("{}h", hours);
        }
        let minutes = dur.num_minutes();
        format!("{}m", minutes.max(0))
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UpdateType {
    Major,
    Minor,
    Patch,
    Digest,
    Pin,
    Unknown,
}

impl UpdateType {
    /// Sort priority for merge ordering: lower = merged first.
    pub fn merge_order(self) -> u8 {
        match self {
            UpdateType::Pin => 0,
            UpdateType::Digest => 1,
            UpdateType::Patch => 2,
            UpdateType::Minor => 3,
            UpdateType::Major => 4,
            UpdateType::Unknown => 5,
        }
    }

    /// Classify from labels first, then title, defaulting to Unknown.
    pub fn classify(labels: &[String], title: &str) -> Self {
        if let Some(ut) = Self::classify_labels(labels) {
            return ut;
        }
        if let Some(ut) = Self::classify_title(title) {
            return ut;
        }
        UpdateType::Unknown
    }

    /// Substring match (case-insensitive) across labels.
    pub fn classify_labels(labels: &[String]) -> Option<Self> {
        for label in labels {
            let lower = label.to_lowercase();
            if lower.contains("major") {
                return Some(UpdateType::Major);
            }
            if lower.contains("minor") {
                return Some(UpdateType::Minor);
            }
            if lower.contains("patch") {
                return Some(UpdateType::Patch);
            }
            if lower.contains("digest") {
                return Some(UpdateType::Digest);
            }
            if lower.contains("pin") {
                return Some(UpdateType::Pin);
            }
        }
        None
    }

    /// Look for "(major)", "(minor)", etc. in the title.
    pub fn classify_title(title: &str) -> Option<Self> {
        let lower = title.to_lowercase();
        if lower.contains("(major)") {
            return Some(UpdateType::Major);
        }
        if lower.contains("(minor)") {
            return Some(UpdateType::Minor);
        }
        if lower.contains("(patch)") {
            return Some(UpdateType::Patch);
        }
        if lower.contains("(digest)") {
            return Some(UpdateType::Digest);
        }
        if lower.contains("(pin)") {
            return Some(UpdateType::Pin);
        }
        None
    }
}

impl fmt::Display for UpdateType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            UpdateType::Major => write!(f, "major"),
            UpdateType::Minor => write!(f, "minor"),
            UpdateType::Patch => write!(f, "patch"),
            UpdateType::Digest => write!(f, "digest"),
            UpdateType::Pin => write!(f, "pin"),
            UpdateType::Unknown => write!(f, "\u{2014}"),
        }
    }
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct SystemStatus {
    pub version: String,
    pub boot_time: DateTime<Utc>,
    pub uptime: String,
    pub queue_size: u64,
    pub running_jobs: u64,
    pub failed_jobs: u64,
    pub last_finished: Option<Job>,
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
pub struct Job {
    pub id: String,
    pub repo: String,
    pub state: JobState,
    pub started_at: Option<DateTime<Utc>>,
    pub duration: Option<Duration>,
    pub trigger: Option<String>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum JobState {
    Running,
    Pending,
    Finished,
    Failed,
}

impl fmt::Display for JobState {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            JobState::Running => write!(f, "running"),
            JobState::Pending => write!(f, "pending"),
            JobState::Finished => write!(f, "finished"),
            JobState::Failed => write!(f, "failed"),
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Panel {
    Sidebar,
    PrTable,
    Detail,
}

impl Panel {
    pub fn next(self) -> Self {
        match self {
            Panel::Sidebar => Panel::PrTable,
            Panel::PrTable => Panel::Detail,
            Panel::Detail => Panel::Sidebar,
        }
    }

    pub fn prev(self) -> Self {
        match self {
            Panel::Sidebar => Panel::Detail,
            Panel::PrTable => Panel::Sidebar,
            Panel::Detail => Panel::PrTable,
        }
    }
}

#[derive(Debug, Clone)]
pub enum ConfirmAction {
    MergePr(u64, String),
    MergeAllSafe(String),
    MergeAll(String),
    ClosePr(u64, String),
    PurgeJobs,
}

impl fmt::Display for ConfirmAction {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ConfirmAction::MergePr(num, repo) => {
                write!(f, "Merge PR #{} in {}?", num, repo)
            }
            ConfirmAction::MergeAllSafe(repo) => {
                write!(f, "Merge all safe PRs in {}?", repo)
            }
            ConfirmAction::MergeAll(repo) => {
                write!(f, "Merge ALL PRs in {}? (including major)", repo)
            }
            ConfirmAction::ClosePr(num, repo) => {
                write!(f, "Close PR #{} in {}?", num, repo)
            }
            ConfirmAction::PurgeJobs => write!(f, "Purge all finished jobs?"),
        }
    }
}

#[derive(Debug, Clone)]
pub struct FlashMessage {
    pub text: String,
    pub level: FlashLevel,
    pub expires: Instant,
}

impl FlashMessage {
    pub fn success(text: impl Into<String>) -> Self {
        Self {
            text: text.into(),
            level: FlashLevel::Success,
            expires: Instant::now() + Duration::from_secs(5),
        }
    }

    pub fn error(text: impl Into<String>) -> Self {
        Self {
            text: text.into(),
            level: FlashLevel::Error,
            expires: Instant::now() + Duration::from_secs(5),
        }
    }

    pub fn is_expired(&self) -> bool {
        Instant::now() >= self.expires
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FlashLevel {
    Success,
    Error,
}

pub enum ActionResult {
    PrMerged { repo: String, number: u64 },
    PrClosed { repo: String, number: u64 },
    AllSafeMerged { repo: String, count: usize, skipped: usize },
    AllMerged { repo: String, count: usize, skipped: usize },
    SyncTriggered,
    JobsPurged,
    Error(String),
}

pub struct FetchResult {
    pub repos: anyhow::Result<Vec<Repo>>,
    pub prs: anyhow::Result<HashMap<String, Vec<PR>>>,
    pub status: anyhow::Result<SystemStatus>,
    pub jobs: anyhow::Result<Vec<Job>>,
}

#[cfg(test)]
mod tests {
    use super::*;

    // UpdateType classification from labels
    #[test]
    fn classify_from_label_renovate_colon() {
        assert_eq!(
            UpdateType::classify_labels(&["renovate:major".into()]),
            Some(UpdateType::Major)
        );
    }

    #[test]
    fn classify_from_label_substring() {
        assert_eq!(
            UpdateType::classify_labels(&["update-type:minor".into()]),
            Some(UpdateType::Minor)
        );
    }

    #[test]
    fn classify_from_label_case_insensitive() {
        assert_eq!(
            UpdateType::classify_labels(&["PATCH".into()]),
            Some(UpdateType::Patch)
        );
    }

    #[test]
    fn classify_from_label_no_match() {
        assert_eq!(
            UpdateType::classify_labels(&["bug".into(), "enhancement".into()]),
            None
        );
    }

    #[test]
    fn classify_from_title_major() {
        assert_eq!(
            UpdateType::classify_title("Update react (major)"),
            Some(UpdateType::Major)
        );
    }

    #[test]
    fn classify_from_title_no_match() {
        assert_eq!(UpdateType::classify_title("Update dependencies"), None);
    }

    #[test]
    fn classify_labels_preferred_over_title() {
        assert_eq!(
            UpdateType::classify(&["renovate:minor".into()], "Update react (major)"),
            UpdateType::Minor
        );
    }

    #[test]
    fn classify_falls_back_to_title() {
        assert_eq!(
            UpdateType::classify(&[], "Update react (patch)"),
            UpdateType::Patch
        );
    }

    #[test]
    fn classify_defaults_to_unknown() {
        assert_eq!(
            UpdateType::classify(&[], "Update react"),
            UpdateType::Unknown
        );
    }

    // PR::is_safe
    #[test]
    fn safe_pr_minor_mergeable_checks_pass() {
        let pr = make_pr(UpdateType::Minor, Some(true), Some(true));
        assert!(pr.is_safe());
    }

    #[test]
    fn safe_pr_patch_mergeable_checks_pass() {
        let pr = make_pr(UpdateType::Patch, Some(true), Some(true));
        assert!(pr.is_safe());
    }

    #[test]
    fn unsafe_pr_major() {
        let pr = make_pr(UpdateType::Major, Some(true), Some(true));
        assert!(!pr.is_safe());
    }

    #[test]
    fn unsafe_pr_not_mergeable() {
        let pr = make_pr(UpdateType::Minor, Some(false), Some(true));
        assert!(!pr.is_safe());
    }

    #[test]
    fn unsafe_pr_checks_fail() {
        let pr = make_pr(UpdateType::Minor, Some(true), Some(false));
        assert!(!pr.is_safe());
    }

    #[test]
    fn unsafe_pr_mergeable_unknown() {
        let pr = make_pr(UpdateType::Minor, None, Some(true));
        assert!(!pr.is_safe());
    }

    fn make_pr(update_type: UpdateType, mergeable: Option<bool>, checks_pass: Option<bool>) -> PR {
        PR {
            number: 1,
            title: "test".into(),
            repo: "org/repo".into(),
            branch: "renovate/test".into(),
            base: "main".into(),
            url: "https://github.com/org/repo/pull/1".into(),
            created_at: chrono::Utc::now(),
            update_type,
            mergeable,
            checks_pass,
        }
    }
}
