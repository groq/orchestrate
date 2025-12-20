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
        if self.command.chars().count() > 30 {
            format!("{}...", self.command.chars().take(27).collect::<String>())
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
        if self.n <= 0 {
            1
        } else {
            self.n
        }
    }

    pub fn is_valid(&self) -> bool {
        !self.agent.is_empty()
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
    pub error: Option<String>,
}

pub fn load(dir: &Path) -> LoadResult {
    let path = dir.join(SETTINGS_FILE_NAME);
    if !path.exists() {
        return LoadResult {
            config: None,
            path: None,
            error: None,
        };
    }

    let data = match fs::read_to_string(&path) {
        Ok(d) => d,
        Err(e) => {
            return LoadResult {
                config: None,
                path: Some(path),
                error: Some(format!("failed to read file: {}", e)),
            };
        }
    };

    let config: Config = match serde_yaml::from_str(&data) {
        Ok(c) => c,
        Err(e) => {
            return LoadResult {
                config: None,
                path: Some(path),
                error: Some(format!("failed to parse YAML: {}", e)),
            };
        }
    };

    LoadResult {
        config: Some(config),
        path: Some(path),
        error: None,
    }
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
    // Use as_bytes() to safely handle multi-byte UTF-8 characters.
    // Hex colors must be exactly 6 ASCII bytes (0-9, a-f, A-F).
    let bytes = s.as_bytes();
    if bytes.len() != 6 || !bytes.iter().all(|b| b.is_ascii_hexdigit()) {
        return None;
    }
    // Safe to use string slicing now since we verified all bytes are ASCII hex digits
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
        // Multi-byte UTF-8 that happens to be 6 bytes should not panic
        assert!(parse_hex_color("€€").is_none()); // 2 euro signs = 6 bytes
    }
}
