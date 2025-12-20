use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Command {
    #[serde(default)]
    pub command: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub color: String,
}

impl Command {
    pub fn display_title(&self) -> String {
        if !self.title.is_empty() {
            return self.title.clone();
        }
        if self.command.is_empty() {
            return "terminal".to_string();
        }
        if self.command.len() > 30 {
            format!("{}...", &self.command[..27])
        } else {
            self.command.clone()
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Worktree {
    #[serde(default)]
    pub agent: String,
    #[serde(default)]
    pub n: i64,
    #[serde(default)]
    pub commands: Vec<Command>,
}

impl Worktree {
    pub fn get_n(&self) -> i64 {
        if self.n <= 0 { 1 } else { self.n }
    }

    pub fn is_valid(&self) -> bool {
        !self.agent.is_empty()
    }

    pub fn has_commands(&self) -> bool {
        !self.commands.is_empty()
    }
}

pub type Preset = Vec<Worktree>;

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Config {
    #[serde(default)]
    pub default: String,
    #[serde(default)]
    pub presets: HashMap<String, Preset>,
}

pub const SETTINGS_FILE_NAME: &str = "settings.yaml";

#[derive(Debug)]
pub struct LoadResult {
    pub config: Option<Config>,
    pub path: Option<PathBuf>,
}

pub fn load(dir: &Path) -> LoadResult {
    let path = dir.join(SETTINGS_FILE_NAME);
    if !path.exists() {
        return LoadResult {
            config: None,
            path: None,
        };
    }

    let data = match fs::read_to_string(&path) {
        Ok(d) => d,
        Err(_) => {
            return LoadResult {
                config: None,
                path: None,
            };
        }
    };

    let config: Config = match serde_yaml::from_str(&data) {
        Ok(c) => c,
        Err(_) => {
            return LoadResult {
                config: None,
                path: None,
            };
        }
    };

    LoadResult {
        config: Some(config),
        path: Some(path),
    }
}

pub fn save_preset_config(dir: &Path, cfg: &Config) -> Result<()> {
    let path = dir.join(SETTINGS_FILE_NAME);
    let yaml = serde_yaml::to_string(cfg)?;
    fs::create_dir_all(dir)?;
    fs::write(&path, yaml).with_context(|| format!("failed writing {}", path.display()))?;
    Ok(())
}

pub fn get_preset(cfg: &Config, name: &str) -> Option<Preset> {
    cfg.presets.get(name).cloned()
}

pub fn get_default_preset_name(cfg: &Config, app_default: &str) -> String {
    if !cfg.default.is_empty() {
        cfg.default.clone()
    } else {
        app_default.to_string()
    }
}

/// Parse a hex color string (e.g., "#ff8c00") into RGB values.
pub fn parse_hex_color(hex: &str) -> Option<(u8, u8, u8)> {
    let mut s = hex.trim();
    if s.starts_with('#') {
        s = &s[1..];
    }
    if s.len() != 6 {
        return None;
    }
    let r = u8::from_str_radix(&s[0..2], 16).ok()?;
    let g = u8::from_str_radix(&s[2..4], 16).ok()?;
    let b = u8::from_str_radix(&s[4..6], 16).ok()?;
    Some((r, g, b))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_hex_color_valid() {
        assert_eq!(parse_hex_color("#ff8800"), Some((255, 136, 0)));
        assert_eq!(parse_hex_color("00ff00"), Some((0, 255, 0)));
    }

    #[test]
    fn parse_hex_color_invalid() {
        assert!(parse_hex_color("abc").is_none());
        assert!(parse_hex_color("#12345").is_none());
    }
}
