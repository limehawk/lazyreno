use std::path::{Path, PathBuf};
use std::process::Command;
use std::time::Duration;

use anyhow::{Context, Result};
use serde::Deserialize;

#[derive(Debug, Deserialize)]
pub struct Config {
    pub renovate: RenovateConfig,
    pub github: GithubConfig,
    #[serde(default)]
    pub ui: UiConfig,
}

#[derive(Debug, Deserialize)]
pub struct RenovateConfig {
    pub url: String,
    pub secret: String,
}

#[derive(Debug, Deserialize)]
pub struct GithubConfig {
    pub owner: String,
    pub token: String,
}

#[derive(Debug, Deserialize)]
pub struct UiConfig {
    #[serde(
        default = "default_refresh_interval",
        deserialize_with = "humantime_deser"
    )]
    pub refresh_interval: Duration,
    #[serde(default = "default_accent")]
    pub accent: String,
}

impl Default for UiConfig {
    fn default() -> Self {
        Self {
            refresh_interval: default_refresh_interval(),
            accent: default_accent(),
        }
    }
}

fn default_refresh_interval() -> Duration {
    Duration::from_secs(30)
}

fn default_accent() -> String {
    "cyan".to_string()
}

fn humantime_deser<'de, D>(deserializer: D) -> Result<Duration, D::Error>
where
    D: serde::Deserializer<'de>,
{
    let s = String::deserialize(deserializer)?;
    humantime::parse_duration(&s).map_err(serde::de::Error::custom)
}

impl Config {
    /// Default config path: ~/.config/lazyreno/config.toml
    pub fn default_path() -> PathBuf {
        dirs::config_dir()
            .unwrap_or_else(|| PathBuf::from("~/.config"))
            .join("lazyreno")
            .join("config.toml")
    }

    /// Load config from a TOML file, then apply env var overrides
    /// and resolve any op:// references.
    pub fn load_from_path(path: &Path) -> Result<Self> {
        let content =
            std::fs::read_to_string(path).with_context(|| format!("reading {}", path.display()))?;
        let mut cfg: Config =
            toml::from_str(&content).with_context(|| format!("parsing {}", path.display()))?;

        // Env var overrides
        if let Ok(val) = std::env::var("LAZYRENO_RENOVATE_URL") {
            cfg.renovate.url = val;
        }
        if let Ok(val) = std::env::var("LAZYRENO_RENOVATE_SECRET") {
            cfg.renovate.secret = val;
        }
        if let Ok(val) = std::env::var("LAZYRENO_GITHUB_TOKEN") {
            cfg.github.token = val;
        } else if let Ok(val) = std::env::var("GITHUB_TOKEN") {
            cfg.github.token = val;
        }
        if let Ok(val) = std::env::var("LAZYRENO_GITHUB_OWNER") {
            cfg.github.owner = val;
        }

        // Resolve op:// references
        if is_op_reference(&cfg.renovate.url) {
            cfg.renovate.url = resolve_op(&cfg.renovate.url)?;
        }
        if is_op_reference(&cfg.renovate.secret) {
            cfg.renovate.secret = resolve_op(&cfg.renovate.secret)?;
        }
        if is_op_reference(&cfg.github.token) {
            cfg.github.token = resolve_op(&cfg.github.token)?;
        }

        Ok(cfg)
    }
}

/// Returns true if the value is a 1Password secret reference.
pub fn is_op_reference(val: &str) -> bool {
    val.starts_with("op://")
}

/// Resolves a 1Password secret reference by shelling out to `op read`.
pub fn resolve_op(val: &str) -> Result<String> {
    let output = Command::new("op")
        .args(["read", val])
        .output()
        .with_context(|| format!("failed to run `op read {}`", val))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        anyhow::bail!("op read {} failed: {}", val, stderr.trim());
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    fn write_config(content: &str) -> NamedTempFile {
        let mut f = NamedTempFile::new().unwrap();
        f.write_all(content.as_bytes()).unwrap();
        f
    }

    #[test]
    fn load_minimal_config() {
        let f = write_config(
            r#"
            [renovate]
            url = "https://renovate.example.com"
            secret = "test-secret"
            [github]
            owner = "my-org"
            token = "ghp_test"
        "#,
        );
        let cfg = Config::load_from_path(f.path()).unwrap();
        assert_eq!(cfg.renovate.url, "https://renovate.example.com");
        assert_eq!(cfg.github.owner, "my-org");
        assert_eq!(cfg.ui.refresh_interval, Duration::from_secs(30));
        assert_eq!(cfg.ui.accent, "cyan");
    }

    #[test]
    fn load_with_ui_overrides() {
        let f = write_config(
            r#"
            [renovate]
            url = "https://renovate.example.com"
            secret = "test-secret"
            [github]
            owner = "my-org"
            token = "ghp_test"
            [ui]
            refresh_interval = "5m"
            accent = "magenta"
        "#,
        );
        let cfg = Config::load_from_path(f.path()).unwrap();
        assert_eq!(cfg.ui.refresh_interval, Duration::from_secs(300));
        assert_eq!(cfg.ui.accent, "magenta");
    }

    #[test]
    fn missing_required_field_errors() {
        let f = write_config(
            r#"
            [renovate]
            url = "https://renovate.example.com"
            [github]
            owner = "my-org"
            token = "ghp_test"
        "#,
        );
        assert!(Config::load_from_path(f.path()).is_err());
    }

    #[test]
    fn is_op_ref() {
        assert!(is_op_reference("op://Dev/item/field"));
        assert!(!is_op_reference("ghp_test123"));
    }

    #[test]
    fn env_override_token() {
        let f = write_config(
            r#"
            [renovate]
            url = "https://renovate.example.com"
            secret = "test-secret"
            [github]
            owner = "my-org"
            token = "ghp_test"
        "#,
        );
        unsafe { std::env::set_var("LAZYRENO_GITHUB_TOKEN", "env-token") };
        let cfg = Config::load_from_path(f.path()).unwrap();
        assert_eq!(cfg.github.token, "env-token");
        unsafe { std::env::remove_var("LAZYRENO_GITHUB_TOKEN") };
    }
}
