use anyhow::{anyhow, Context, Result};
use std::fs;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::str;

#[derive(Debug, Clone)]
pub struct FileStats {
    pub path: String,
    pub adds: i32,
    pub deletes: i32,
}

fn run_git(dir: &Path, args: &[&str]) -> Result<String> {
    let output = Command::new("git")
        .args(args)
        .current_dir(dir)
        .output()
        .with_context(|| format!("failed to run git {:?}", args))?;

    if !output.status.success() {
        return Err(anyhow!(
            "git {:?} failed: {}",
            args,
            String::from_utf8_lossy(&output.stderr)
        ));
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

pub fn ensure_repo(repo_spec: &str, base_dir: &Path) -> Result<PathBuf> {
    let parts: Vec<&str> = repo_spec.split('/').collect();
    if parts.len() != 2 {
        return Err(anyhow!("invalid repo format (expected owner/repo)"));
    }
    let owner = parts[0];
    let repo = parts[1];
    let repo_url = format!("https://github.com/{}/{}.git", owner, repo);

    fs::create_dir_all(base_dir)?;
    let repo_path = base_dir.join(format!("{}-{}", owner, repo));

    if repo_path.join(".git").exists() {
        // In test mode, skip fetch/reset since we use local repos without remotes
        if std::env::var("DISPATCH_TEST_MODE").is_err() {
            fetch_and_reset(&repo_path)?;
        }
        return Ok(repo_path);
    }

    let output = Command::new("git")
        .args(["clone", &repo_url, repo_path.to_str().unwrap()])
        .output()
        .with_context(|| format!("failed to clone {}", repo_url))?;

    if !output.status.success() {
        return Err(anyhow!(
            "git clone failed: {}",
            String::from_utf8_lossy(&output.stderr)
        ));
    }

    Ok(repo_path)
}

pub fn fetch_and_reset(repo_path: &Path) -> Result<()> {
    run_git(repo_path, &["fetch", "origin", "main"])?;
    run_git(repo_path, &["reset", "--hard", "origin/main"])?;
    run_git(repo_path, &["clean", "-fd"])?;
    Ok(())
}

pub fn create_worktree(
    repo_root: &Path,
    worktree_path: &Path,
    branch_name: &str,
    base_branch: &str,
) -> Result<()> {
    if worktree_path.exists() {
        return Err(anyhow!(
            "worktree path already exists: {}",
            worktree_path.display()
        ));
    }

    run_git(
        repo_root,
        &[
            "worktree",
            "add",
            "-b",
            branch_name,
            worktree_path.to_str().unwrap(),
            base_branch,
        ],
    )?;
    Ok(())
}

pub fn get_status_stats(path: &Path) -> Result<(i32, i32)> {
    let out = run_git(path, &["diff", "HEAD", "--numstat"])?;
    let mut adds = 0;
    let mut deletes = 0;
    for line in out.lines() {
        if line.trim().is_empty() {
            continue;
        }
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() >= 2 {
            adds += parts[0].parse::<i32>().unwrap_or(0);
            deletes += parts[1].parse::<i32>().unwrap_or(0);
        }
    }
    Ok((adds, deletes))
}

pub fn get_detailed_status_stats(path: &Path) -> Result<Vec<FileStats>> {
    let out = run_git(path, &["diff", "HEAD", "--numstat"])?;
    let mut stats = Vec::new();
    for line in out.lines() {
        if line.trim().is_empty() {
            continue;
        }
        let parts: Vec<&str> = line.split('\t').collect();
        if parts.len() >= 3 {
            let adds = parts[0].parse::<i32>().unwrap_or(0);
            let deletes = parts[1].parse::<i32>().unwrap_or(0);
            stats.push(FileStats {
                path: parts[2].to_string(),
                adds,
                deletes,
            });
        }
    }
    Ok(stats)
}

pub fn current_branch(path: &Path) -> Result<String> {
    run_git(path, &["rev-parse", "--abbrev-ref", "HEAD"])
}

pub fn last_commit_time(path: &Path) -> Result<String> {
    run_git(path, &["log", "-1", "--format=%cr"])
}

pub fn remote_url(path: &Path) -> Result<String> {
    run_git(path, &["remote", "get-url", "origin"]).map(|u| u.trim_end_matches(".git").to_string())
}

pub fn recent_commits(path: &Path, n: usize) -> Result<Vec<String>> {
    let out = run_git(
        path,
        &["log", "-n", &n.to_string(), "--pretty=format:%h %cr %s"],
    )?;
    let lines = out
        .lines()
        .map(|l| l.trim().to_string())
        .filter(|l| !l.is_empty())
        .collect();
    Ok(lines)
}
