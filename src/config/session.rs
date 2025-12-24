use anyhow::{Context, Result};
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::Path;

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SessionMetadata {
    #[serde(default = "Utc::now")]
    pub created_at: DateTime<Utc>,
    #[serde(default)]
    pub repo: String,
    #[serde(default)]
    pub branch: String,
    #[serde(default)]
    pub prompt: String,
    #[serde(default)]
    pub preset_name: String,
    #[serde(default)]
    pub agents: Vec<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub last_opened: Option<DateTime<Utc>>,
}

pub const SESSION_METADATA_FILE: &str = ".dispatch-session.yaml";

pub fn create_session_metadata(
    repo: &str,
    branch: &str,
    prompt: &str,
    preset_name: &str,
    agents: &[String],
) -> SessionMetadata {
    SessionMetadata {
        created_at: Utc::now(),
        repo: repo.to_string(),
        branch: branch.to_string(),
        prompt: prompt.to_string(),
        preset_name: preset_name.to_string(),
        agents: agents.to_vec(),
        last_opened: None,
    }
}

pub fn load_session_metadata(worktree_path: &Path) -> Result<SessionMetadata> {
    let path = worktree_path.join(SESSION_METADATA_FILE);
    let contents = fs::read_to_string(&path)
        .with_context(|| format!("failed to read session metadata {}", path.display()))?;
    let meta: SessionMetadata = serde_yaml::from_str(&contents)
        .with_context(|| format!("failed to parse session metadata {}", path.display()))?;
    Ok(meta)
}

pub fn save_session_metadata(worktree_path: &Path, meta: &SessionMetadata) -> Result<()> {
    let path = worktree_path.join(SESSION_METADATA_FILE);
    let yaml = serde_yaml::to_string(meta)?;
    fs::write(&path, yaml)
        .with_context(|| format!("failed to write session metadata {}", path.display()))?;
    Ok(())
}
