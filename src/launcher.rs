use crate::config::preset::{self, Preset, Worktree};
use crate::config::session;
use crate::git;
use crate::terminal::{SessionInfo, TerminalManager};
use crate::util;
use anyhow::{anyhow, Context, Result};
use std::fs;
use std::path::PathBuf;

#[derive(Debug, Clone)]
pub struct Options {
    pub repo: String,
    pub name: String,
    pub prompt: String,
    pub preset_name: String,
    pub multiplier: i64,
    pub data_dir: PathBuf,
    pub preset: Preset,
    pub maximize_on_launch: bool,
}

#[derive(Debug)]
pub struct LaunchResult {
    pub sessions: Vec<SessionInfo>,
    pub terminal_window_count: usize,
}

pub fn launch(opts: Options) -> Result<LaunchResult> {
    validate_repo(&opts.repo)?;
    validate_name(&opts.name)?;
    validate_prompt(&opts.prompt)?;

    let repos_dir = opts.data_dir.join("repos");
    let worktrees_dir = opts.data_dir.join("worktrees");
    fs::create_dir_all(&worktrees_dir)?;

    let repo_root = git::ensure_repo(&opts.repo, &repos_dir)
        .with_context(|| format!("failed to ensure repo {}", opts.repo))?;
    let base_branch = "main";

    let mut sessions: Vec<SessionInfo> = Vec::new();
    let preset = if opts.preset.is_empty() {
        vec![Worktree {
            agent: "claude".to_string(),
            n: 1,
            commands: vec![],
        }]
    } else {
        opts.preset.clone()
    };

    for w in preset {
        if !w.is_valid() {
            continue;
        }
        let effective_n = calculate_effective_n(w.n, opts.multiplier);

        for _ in 0..effective_n {
            let suffix = util::random_hex(4);
            let branch_name = format!("{}-{}", opts.name, suffix);
            let worktree_path = worktrees_dir.join(format!(
                "{}-{}",
                repo_root.file_name().unwrap().to_string_lossy(),
                branch_name
            ));
            let activity_path = opts
                .data_dir
                .join("activity")
                .join(format!("{}.log", branch_name));

            if let Err(err) =
                git::create_worktree(&repo_root, &worktree_path, &branch_name, base_branch)
            {
                eprintln!(
                    "Warning: failed to create worktree {}: {}",
                    branch_name, err
                );
                continue;
            }

            let meta = session::create_session_metadata(
                &opts.repo,
                &branch_name,
                &opts.prompt,
                &opts.preset_name,
                std::slice::from_ref(&w.agent),
            );
            let _ = session::save_session_metadata(&worktree_path, &meta);

            let worktree_path_str = worktree_path.to_string_lossy().to_string();
            let mut agent_session =
                SessionInfo::agent_session(&worktree_path_str, &branch_name, &w.agent);
            agent_session.activity_log = Some(activity_path.to_string_lossy().to_string());
            sessions.push(agent_session);

            for cmd in &w.commands {
                let color = preset::parse_hex_color(&cmd.color);
                sessions.push(SessionInfo::custom_command(
                    &cmd.command,
                    &cmd.display_title(),
                    color,
                    &worktree_path_str,
                    &branch_name,
                ));
            }
        }
    }

    if sessions.is_empty() {
        return Err(anyhow!("no sessions were created"));
    }

    let mgr = TerminalManager::new(opts.maximize_on_launch);
    let window_count = mgr.launch_sessions(&sessions, &opts.prompt)?;

    Ok(LaunchResult {
        sessions,
        terminal_window_count: window_count,
    })
}

/// Calculate the effective number of instances to create for a worktree.
/// If multiplier > 0, it overrides the preset's n value.
/// If multiplier <= 0, the preset's n value is used (via worktree.get_n()).
pub fn calculate_effective_n(worktree_n: i64, multiplier: i64) -> i64 {
    let base_n = if worktree_n <= 0 { 1 } else { worktree_n };
    if multiplier > 0 {
        multiplier
    } else {
        base_n
    }
}

pub fn validate_repo(repo: &str) -> Result<()> {
    if repo.is_empty() {
        return Err(anyhow!("repository is required"));
    }
    if repo.split('/').count() != 2 {
        return Err(anyhow!("invalid repository format, expected 'owner/repo'"));
    }
    Ok(())
}

pub fn validate_name(name: &str) -> Result<()> {
    if name.is_empty() {
        return Err(anyhow!("branch name prefix is required"));
    }
    // Reject characters that could break shell commands or AppleScript
    let forbidden = [' ', '/', '\\', '\'', '"', '`', '$'];
    if name.chars().any(|c| forbidden.contains(&c)) {
        return Err(anyhow!(
            "branch name cannot contain spaces, slashes, quotes, backticks, or dollar signs"
        ));
    }
    Ok(())
}

pub fn validate_prompt(prompt: &str) -> Result<()> {
    if prompt.is_empty() {
        return Err(anyhow!("prompt is required"));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validate_repo_accepts_owner_repo() {
        assert!(validate_repo("owner/repo").is_ok());
    }

    #[test]
    fn validate_repo_rejects_invalid() {
        assert!(validate_repo("invalid").is_err());
    }

    #[test]
    fn validate_name_disallows_invalid_chars() {
        assert!(validate_name("good-name").is_ok());
        assert!(validate_name("also_good123").is_ok());
        assert!(validate_name("bad name").is_err());
        assert!(validate_name("bad'quote").is_err());
        assert!(validate_name("bad\"quote").is_err());
        assert!(validate_name("bad`tick").is_err());
        assert!(validate_name("bad$dollar").is_err());
        assert!(validate_name("bad/slash").is_err());
        assert!(validate_name("bad\\backslash").is_err());
    }

    #[test]
    fn validate_prompt_requires_content() {
        assert!(validate_prompt("hi").is_ok());
        assert!(validate_prompt("").is_err());
    }

    // ==================== Effective N Calculation Tests ====================
    // These tests verify the multiplier vs preset n logic that determines
    // how many instances of each worktree agent are created.

    mod effective_n {
        use super::*;

        #[test]
        fn multiplier_zero_uses_preset_n() {
            // When multiplier is 0, the preset's n value should be used.
            // This is the expected behavior when launching from TUI without override.
            assert_eq!(calculate_effective_n(3, 0), 3);
            assert_eq!(calculate_effective_n(5, 0), 5);
            assert_eq!(calculate_effective_n(1, 0), 1);
        }

        #[test]
        fn multiplier_positive_overrides_preset_n() {
            // When multiplier > 0, it should override the preset's n value.
            assert_eq!(calculate_effective_n(3, 2), 2);
            assert_eq!(calculate_effective_n(1, 5), 5);
            assert_eq!(calculate_effective_n(10, 1), 1);
        }

        #[test]
        fn preset_n_zero_or_negative_defaults_to_one() {
            // When preset n is <= 0, it should default to 1.
            assert_eq!(calculate_effective_n(0, 0), 1);
            assert_eq!(calculate_effective_n(-1, 0), 1);
            assert_eq!(calculate_effective_n(-5, 0), 1);
        }

        #[test]
        fn multiplier_overrides_even_invalid_preset_n() {
            // Multiplier should still work even when preset n is invalid.
            assert_eq!(calculate_effective_n(0, 3), 3);
            assert_eq!(calculate_effective_n(-1, 2), 2);
        }

        #[test]
        fn negative_multiplier_uses_preset_n() {
            // Negative multiplier should behave like zero (use preset n).
            assert_eq!(calculate_effective_n(3, -1), 3);
            assert_eq!(calculate_effective_n(5, -10), 5);
        }

        #[test]
        fn tui_launch_should_use_multiplier_zero() {
            // This test documents the expected behavior for TUI launches:
            // TUI should pass multiplier=0 to respect preset n values.
            //
            // A preset with n=2 should create 2 instances when launched from TUI.
            // Previously, TUI hardcoded multiplier=1, which caused this bug.
            let preset_n = 2;
            let tui_multiplier = 0; // TUI should pass 0 to respect preset
            assert_eq!(
                calculate_effective_n(preset_n, tui_multiplier),
                2,
                "TUI launch with multiplier=0 should respect preset n=2"
            );

            // This was the bug: TUI was passing multiplier=1
            let buggy_multiplier = 1;
            assert_eq!(
                calculate_effective_n(preset_n, buggy_multiplier),
                1,
                "multiplier=1 overrides preset n (this was the bug)"
            );
        }

        #[test]
        fn parallel_preset_example() {
            // Test the "parallel" preset scenario which has multiple worktrees
            // each with n=2. When launched from TUI, each should create 2 instances.
            let claude_n = 2;
            let codex_n = 2;
            let tui_multiplier = 0;

            assert_eq!(calculate_effective_n(claude_n, tui_multiplier), 2);
            assert_eq!(calculate_effective_n(codex_n, tui_multiplier), 2);
        }
    }
}
