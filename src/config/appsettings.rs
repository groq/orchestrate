use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Default)]
#[serde(rename_all = "lowercase")]
pub enum TerminalType {
    #[default]
    #[serde(rename = "iterm2")]
    ITerm2,
    #[serde(rename = "terminal")]
    Terminal,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TerminalSettings {
    #[serde(default)]
    pub r#type: TerminalType,
    #[serde(default = "default_maximize")]
    pub maximize_on_launch: bool,
}

fn default_maximize() -> bool {
    true
}

impl Default for TerminalSettings {
    fn default() -> Self {
        TerminalSettings {
            r#type: TerminalType::ITerm2,
            maximize_on_launch: true,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UISettings {
    #[serde(default = "default_show_status_bar")]
    pub show_status_bar: bool,
}

fn default_show_status_bar() -> bool {
    true
}

impl Default for UISettings {
    fn default() -> Self {
        UISettings {
            show_status_bar: true,
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionSettings {
    #[serde(default = "default_preset_name")]
    pub default_preset: String,
    #[serde(default)]
    pub auto_clean_worktrees: bool,
    #[serde(default = "default_retention_days")]
    pub worktree_retention_days: i64,
}

fn default_preset_name() -> String {
    "default".to_string()
}

fn default_retention_days() -> i64 {
    7
}

impl Default for SessionSettings {
    fn default() -> Self {
        SessionSettings {
            default_preset: default_preset_name(),
            auto_clean_worktrees: false,
            worktree_retention_days: default_retention_days(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AppSettings {
    #[serde(default)]
    pub terminal: TerminalSettings,
    #[serde(default)]
    pub ui: UISettings,
    #[serde(default)]
    pub session: SessionSettings,
}

impl Default for AppSettings {
    fn default() -> Self {
        AppSettings {
            terminal: TerminalSettings {
                r#type: TerminalType::ITerm2,
                maximize_on_launch: true,
            },
            ui: UISettings {
                show_status_bar: true,
            },
            session: SessionSettings {
                default_preset: default_preset_name(),
                auto_clean_worktrees: false,
                worktree_retention_days: default_retention_days(),
            },
        }
    }
}

pub const APP_SETTINGS_FILE: &str = "orchestrate.yaml";

pub fn default_app_settings() -> AppSettings {
    AppSettings::default()
}

pub fn load_app_settings(dir: &Path) -> Result<(AppSettings, PathBuf)> {
    let path = dir.join(APP_SETTINGS_FILE);
    if !path.exists() {
        return Ok((default_app_settings(), path));
    }

    let contents = fs::read_to_string(&path)
        .with_context(|| format!("failed reading app settings {}", path.display()))?;
    let mut settings: AppSettings = serde_yaml::from_str(&contents)
        .with_context(|| format!("failed parsing app settings {}", path.display()))?;
    // Ensure defaults for missing fields
    let defaults = default_app_settings();
    if settings.session.default_preset.is_empty() {
        settings.session.default_preset = defaults.session.default_preset;
    }
    Ok((settings, path))
}

pub fn save_app_settings(dir: &Path, settings: &AppSettings) -> Result<()> {
    let path = dir.join(APP_SETTINGS_FILE);
    let yaml = serde_yaml::to_string(settings)?;
    let header = "# Orchestrate App Settings\n# This file is auto-generated. Edit carefully.\n\n";
    fs::create_dir_all(dir)?;
    fs::write(&path, format!("{}{}", header, yaml))
        .with_context(|| format!("failed writing {}", path.display()))?;
    Ok(())
}
