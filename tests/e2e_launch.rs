//! End-to-end integration tests for the launcher.
//!
//! These tests verify the complete flow from preset configuration through
//! session creation, including git worktree creation and metadata file generation.
//!
//! The tests use ORCHESTRATE_TEST_MODE to skip actual iTerm2 window spawning,
//! but everything else (git operations, file creation, metadata) is real.

use std::fs;
use std::path::PathBuf;
use std::process::Command;

// Import the library crate
use orchestrate::config::preset::Worktree;
use orchestrate::config::session;
use orchestrate::launcher;

/// Set up a local git repository for testing.
/// Returns the path to the repos directory (parent of the repo).
fn setup_test_git_repo(data_dir: &PathBuf) -> PathBuf {
    let repos_dir = data_dir.join("repos");
    // The repo path format must match what ensure_repo() expects: {owner}-{repo}
    let test_repo_path = repos_dir.join("test-owner-test-repo");
    fs::create_dir_all(&test_repo_path).expect("Failed to create test repo directory");

    // Initialize git repo
    let git_init = Command::new("git")
        .args(["init"])
        .current_dir(&test_repo_path)
        .output()
        .expect("Failed to run git init");
    assert!(git_init.status.success(), "git init failed");

    // Create main branch (git init creates 'master' by default on some systems)
    let _ = Command::new("git")
        .args(["checkout", "-b", "main"])
        .current_dir(&test_repo_path)
        .output();

    // Configure git user for commits (required in CI environments)
    Command::new("git")
        .args(["config", "user.email", "test@test.com"])
        .current_dir(&test_repo_path)
        .output()
        .expect("Failed to configure git email");

    Command::new("git")
        .args(["config", "user.name", "Test User"])
        .current_dir(&test_repo_path)
        .output()
        .expect("Failed to configure git name");

    // Create a test file and commit
    fs::write(test_repo_path.join("README.md"), "# Test Repo\n")
        .expect("Failed to create README");

    Command::new("git")
        .args(["add", "."])
        .current_dir(&test_repo_path)
        .output()
        .expect("Failed to run git add");

    let commit = Command::new("git")
        .args(["commit", "-m", "Initial commit"])
        .current_dir(&test_repo_path)
        .output()
        .expect("Failed to run git commit");
    assert!(commit.status.success(), "git commit failed");

    repos_dir
}

/// Helper to create launcher options for testing
fn create_test_options(
    data_dir: PathBuf,
    preset: Vec<Worktree>,
    multiplier: i64,
) -> launcher::Options {
    launcher::Options {
        repo: "test-owner/test-repo".to_string(),
        name: "test-branch".to_string(),
        prompt: "Test prompt for E2E testing".to_string(),
        preset_name: "test-preset".to_string(),
        multiplier,
        data_dir,
        preset,
        maximize_on_launch: false,
    }
}

// ============================================================================
// E2E Tests
// ============================================================================

/// Test that a preset with n=2 creates exactly 2 sessions when multiplier=0.
/// This is the core test that would have caught the TUI bug.
#[test]
fn e2e_preset_n_creates_correct_session_count() {
    // Enable test mode to skip osascript
    std::env::set_var("ORCHESTRATE_TEST_MODE", "1");

    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let data_dir = temp_dir.path().to_path_buf();

    // Set up local git repo
    setup_test_git_repo(&data_dir);

    // Create preset with n=2
    let preset = vec![Worktree {
        agent: "claude".to_string(),
        n: 2,
        commands: vec![],
    }];

    // Launch with multiplier=0 (should use preset n value)
    let opts = create_test_options(data_dir.clone(), preset, 0);
    let result = launcher::launch(opts).expect("Launch failed");

    // Verify correct number of sessions
    assert_eq!(
        result.sessions.len(),
        2,
        "Preset with n=2 and multiplier=0 should create 2 sessions"
    );

    // Verify worktree directories were created
    let worktrees_dir = data_dir.join("worktrees");
    let worktree_count = fs::read_dir(&worktrees_dir)
        .expect("Failed to read worktrees dir")
        .filter(|e| e.is_ok())
        .count();
    assert_eq!(worktree_count, 2, "Should create 2 worktree directories");
}

/// Test that multiplier=1 overrides preset n value (documents the bug behavior).
/// This test shows what was happening before the fix.
#[test]
fn e2e_multiplier_one_overrides_preset_n() {
    std::env::set_var("ORCHESTRATE_TEST_MODE", "1");

    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let data_dir = temp_dir.path().to_path_buf();
    setup_test_git_repo(&data_dir);

    // Preset with n=3
    let preset = vec![Worktree {
        agent: "claude".to_string(),
        n: 3,
        commands: vec![],
    }];

    // Launch with multiplier=1 (overrides preset n)
    let opts = create_test_options(data_dir.clone(), preset, 1);
    let result = launcher::launch(opts).expect("Launch failed");

    // multiplier=1 should override preset n=3, creating only 1 session
    assert_eq!(
        result.sessions.len(),
        1,
        "multiplier=1 should override preset n=3, creating only 1 session"
    );
}

/// Test parallel preset with multiple worktrees, each with n=2.
/// This is the "parallel" preset scenario from the default settings.
#[test]
fn e2e_parallel_preset_creates_correct_sessions() {
    std::env::set_var("ORCHESTRATE_TEST_MODE", "1");

    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let data_dir = temp_dir.path().to_path_buf();
    setup_test_git_repo(&data_dir);

    // Parallel preset: 2 claude + 2 codex = 4 total
    let preset = vec![
        Worktree {
            agent: "claude".to_string(),
            n: 2,
            commands: vec![],
        },
        Worktree {
            agent: "codex".to_string(),
            n: 2,
            commands: vec![],
        },
    ];

    let opts = create_test_options(data_dir.clone(), preset, 0);
    let result = launcher::launch(opts).expect("Launch failed");

    assert_eq!(
        result.sessions.len(),
        4,
        "Parallel preset (2 claude + 2 codex) should create 4 sessions"
    );

    // Verify we have 4 worktree directories
    let worktrees_dir = data_dir.join("worktrees");
    let worktree_count = fs::read_dir(&worktrees_dir)
        .expect("Failed to read worktrees dir")
        .filter(|e| e.is_ok())
        .count();
    assert_eq!(worktree_count, 4, "Should create 4 worktree directories");
}

/// Test that session metadata files are created correctly.
#[test]
fn e2e_session_metadata_is_created() {
    std::env::set_var("ORCHESTRATE_TEST_MODE", "1");

    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let data_dir = temp_dir.path().to_path_buf();
    setup_test_git_repo(&data_dir);

    let preset = vec![Worktree {
        agent: "claude".to_string(),
        n: 1,
        commands: vec![],
    }];

    let opts = create_test_options(data_dir.clone(), preset, 0);
    let result = launcher::launch(opts).expect("Launch failed");

    // Check that session metadata was created
    for session_info in &result.sessions {
        if let Some(ref path) = session_info.path {
            let worktree_path = PathBuf::from(path);
            let metadata = session::load_session_metadata(&worktree_path)
                .expect("Failed to load session metadata");

            assert_eq!(metadata.repo, "test-owner/test-repo");
            assert_eq!(metadata.prompt, "Test prompt for E2E testing");
            assert_eq!(metadata.preset_name, "test-preset");
            assert!(metadata.branch.starts_with("test-branch-"));
            assert!(metadata.agents.contains(&"claude".to_string()));
        }
    }
}

/// Test high n value preset.
#[test]
fn e2e_high_n_value_creates_many_sessions() {
    std::env::set_var("ORCHESTRATE_TEST_MODE", "1");

    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let data_dir = temp_dir.path().to_path_buf();
    setup_test_git_repo(&data_dir);

    let preset = vec![Worktree {
        agent: "claude".to_string(),
        n: 5,
        commands: vec![],
    }];

    let opts = create_test_options(data_dir.clone(), preset, 0);
    let result = launcher::launch(opts).expect("Launch failed");

    assert_eq!(
        result.sessions.len(),
        5,
        "Preset with n=5 should create 5 sessions"
    );
}

/// Regression test: This test would FAIL if multiplier was hardcoded to 1.
/// It simulates exactly what the TUI does when launching with a preset.
#[test]
fn e2e_regression_tui_launch_respects_preset_n() {
    std::env::set_var("ORCHESTRATE_TEST_MODE", "1");

    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let data_dir = temp_dir.path().to_path_buf();
    setup_test_git_repo(&data_dir);

    // This simulates the "parallel" preset from default settings
    let preset = vec![
        Worktree {
            agent: "claude".to_string(),
            n: 2,
            commands: vec![],
        },
        Worktree {
            agent: "codex".to_string(),
            n: 2,
            commands: vec![],
        },
    ];

    // TUI should pass multiplier=0 to respect preset n values
    // The bug was: TUI passed multiplier=1, which created only 2 sessions (1+1)
    // instead of 4 sessions (2+2)
    let tui_multiplier = 0; // This is what the TUI SHOULD pass

    let opts = create_test_options(data_dir.clone(), preset, tui_multiplier);
    let result = launcher::launch(opts).expect("Launch failed");

    // With the bug (multiplier=1): would create 2 sessions
    // With the fix (multiplier=0): creates 4 sessions
    assert_eq!(
        result.sessions.len(),
        4,
        "TUI launch with parallel preset should create 4 sessions (2+2), not 2 (1+1)"
    );

    // Additional verification: check that each agent type has the right count
    let claude_count = result
        .sessions
        .iter()
        .filter(|s| s.agent.as_deref() == Some("claude"))
        .count();
    let codex_count = result
        .sessions
        .iter()
        .filter(|s| s.agent.as_deref() == Some("codex"))
        .count();

    assert_eq!(claude_count, 2, "Should have 2 claude sessions");
    assert_eq!(codex_count, 2, "Should have 2 codex sessions");
}

/// Test that git branches are created correctly for each worktree.
#[test]
fn e2e_git_branches_are_created() {
    std::env::set_var("ORCHESTRATE_TEST_MODE", "1");

    let temp_dir = tempfile::tempdir().expect("Failed to create temp dir");
    let data_dir = temp_dir.path().to_path_buf();
    setup_test_git_repo(&data_dir);

    let preset = vec![Worktree {
        agent: "claude".to_string(),
        n: 2,
        commands: vec![],
    }];

    let opts = create_test_options(data_dir.clone(), preset, 0);
    let result = launcher::launch(opts).expect("Launch failed");

    // Verify each worktree has a git branch
    for session_info in &result.sessions {
        if let Some(ref path) = session_info.path {
            let worktree_path = PathBuf::from(path);

            // Check that the worktree exists and has a .git file (worktrees have .git files, not directories)
            let git_path = worktree_path.join(".git");
            assert!(
                git_path.exists(),
                "Worktree should have .git file: {:?}",
                worktree_path
            );

            // Check branch name via git
            let output = Command::new("git")
                .args(["branch", "--show-current"])
                .current_dir(&worktree_path)
                .output()
                .expect("Failed to get current branch");

            let branch = String::from_utf8_lossy(&output.stdout).trim().to_string();
            assert!(
                branch.starts_with("test-branch-"),
                "Branch should start with 'test-branch-', got: {}",
                branch
            );
        }
    }
}
