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
        let mut effective_n = w.get_n();
        if opts.multiplier > 0 {
            effective_n = opts.multiplier;
        }

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
}
