use anyhow::Result;
use rand::rngs::OsRng;
use rand::TryRngCore;
use std::env;
use std::path::{Path, PathBuf};

/// Generate a random hex string of `n` bytes (2n hex characters).
pub fn random_hex(n: usize) -> String {
    let mut bytes = vec![0u8; n];
    OsRng.try_fill_bytes(&mut bytes).expect("OS RNG failure");
    hex::encode(bytes)
}

/// Return the platform-appropriate data directory for orchestrate, matching the Go version.
/// - macOS: ~/.orchestrate
/// - Linux: ~/.local/share/orchestrate (or $XDG_DATA_HOME/orchestrate)
/// - Windows: %APPDATA%\\Orchestrate
pub fn data_dir() -> Result<PathBuf> {
    let os = env::consts::OS;
    let base = match os {
        "macos" => dirs_home().map(|h| h.join(".orchestrate")),
        "windows" => {
            if let Ok(appdata) = env::var("APPDATA") {
                Some(PathBuf::from(appdata).join("Orchestrate"))
            } else {
                dirs_home().map(|h| h.join("AppData").join("Roaming").join("Orchestrate"))
            }
        }
        _ => {
            if let Ok(xdg) = env::var("XDG_DATA_HOME") {
                Some(PathBuf::from(xdg).join("orchestrate"))
            } else {
                dirs_home().map(|h| h.join(".local").join("share").join("orchestrate"))
            }
        }
    };

    base.ok_or_else(|| anyhow::anyhow!("could not resolve home directory"))
}

fn dirs_home() -> Option<PathBuf> {
    dirs::home_dir()
}

/// Convert an absolute path into a display-friendly path. On unix, replaces the home directory with "~".
pub fn display_path(path: impl AsRef<Path>) -> String {
    let path = path.as_ref();
    if let Some(home) = dirs_home() {
        if let Ok(rel) = path.strip_prefix(&home) {
            return format!("~/{}", rel.display());
        }
    }
    path.display().to_string()
}

/// Return the last `max_lines` of a file. If the file does not exist, returns empty.
pub fn tail_lines(path: &Path, max_lines: usize) -> Result<Vec<String>> {
    if !path.exists() {
        return Ok(vec![]);
    }
    let contents = std::fs::read_to_string(path)?;
    let mut lines: Vec<&str> = contents.lines().collect();
    if lines.len() > max_lines {
        lines = lines.split_off(lines.len() - max_lines);
    }
    Ok(lines.into_iter().map(|s| s.to_string()).collect())
}

/// Check if a file has been modified within the last `seconds`.
pub fn modified_within(path: &Path, seconds: u64) -> Result<bool> {
    let meta = std::fs::metadata(path)?;
    let modified = meta.modified()?;
    let now = std::time::SystemTime::now();
    Ok(modified >= now - std::time::Duration::from_secs(seconds))
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn random_hex_has_correct_length() {
        let h = random_hex(4);
        assert_eq!(h.len(), 8);
    }

    #[test]
    fn display_path_replaces_home() {
        if let Some(home) = dirs_home() {
            let candidate = home.join("example");
            let disp = display_path(&candidate);
            assert!(disp.starts_with("~/"));
        }
    }

    #[test]
    fn tail_lines_returns_last_lines() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("log.txt");
        fs::write(&path, "a\nb\nc\nd\n").unwrap();
        let lines = tail_lines(&path, 2).unwrap();
        assert_eq!(lines, vec!["c".to_string(), "d".to_string()]);
    }

    #[test]
    fn modified_within_detects_recent() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("file.txt");
        fs::write(&path, "hi").unwrap();
        assert!(modified_within(&path, 5).unwrap());
    }
}
