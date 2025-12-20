// Package git_utils provides git-related operations for orchestrate.
package git_utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Commander interface allows mocking of git commands in tests.
type Commander interface {
	Run(dir string, args ...string) (string, error)
}

// DefaultCommander implements Commander using actual git commands.
type DefaultCommander struct{}

// Run executes a git command with the given arguments.
func (c *DefaultCommander) Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// defaultCmd is the default git commander.
var defaultCmd Commander = &DefaultCommander{}

// SetCommander sets a custom commander (useful for testing).
func SetCommander(cmd Commander) {
	defaultCmd = cmd
}

// ResetCommander resets to the default commander.
func ResetCommander() {
	defaultCmd = &DefaultCommander{}
}

// GetRoot returns the root directory of the current git repository.
func GetRoot() (string, error) {
	return GetRootWithCmd(defaultCmd)
}

// GetRootWithCmd returns the root using a custom commander.
func GetRootWithCmd(cmd Commander) (string, error) {
	out, err := cmd.Run("", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not in a git repository: %w", err)
	}
	return out, nil
}

// GetCurrentBranch returns the current git branch name.
func GetCurrentBranch() (string, error) {
	return GetCurrentBranchWithCmd(defaultCmd)
}

// GetCurrentBranchWithCmd returns the current branch using a custom commander.
func GetCurrentBranchWithCmd(cmd Commander) (string, error) {
	out, err := cmd.Run("", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return out, nil
}

// CreateWorktree creates a new git worktree with a new branch.
func CreateWorktree(repoRoot, worktreePath, branchName, baseBranch string) error {
	return CreateWorktreeWithCmd(defaultCmd, repoRoot, worktreePath, branchName, baseBranch)
}

// CreateWorktreeWithCmd creates a worktree using a custom commander.
func CreateWorktreeWithCmd(cmd Commander, repoRoot, worktreePath, branchName, baseBranch string) error {
	// Check if worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	// Create worktree with new branch based on current branch
	out, err := cmd.Run(repoRoot, "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

// WorktreeExists checks if a path already exists (for worktree creation).
func WorktreeExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// WorktreeInfo contains information about a git worktree.
type WorktreeInfo struct {
	Path       string // Absolute path to the worktree
	Branch     string // Branch name
	Head       string // Commit SHA
	IsBare     bool   // Whether this is the bare worktree
	IsMain     bool   // Whether this is the main worktree
	CreatedAt  string // When the worktree was created (if available)
	LastCommit string // Last commit time
}

// ListWorktrees returns all worktrees for a repository.
func ListWorktrees(repoPath string) ([]WorktreeInfo, error) {
	return ListWorktreesWithCmd(defaultCmd, repoPath)
}

// ListWorktreesWithCmd lists worktrees using a custom commander.
func ListWorktreesWithCmd(cmd Commander, repoPath string) ([]WorktreeInfo, error) {
	// Use porcelain format for easier parsing
	out, err := cmd.Run(repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktrees []WorktreeInfo
	var current *WorktreeInfo

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			current = &WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if current != nil {
			if strings.HasPrefix(line, "HEAD ") {
				current.Head = strings.TrimPrefix(line, "HEAD ")
			} else if strings.HasPrefix(line, "branch ") {
				branch := strings.TrimPrefix(line, "branch ")
				// Strip refs/heads/ prefix
				branch = strings.TrimPrefix(branch, "refs/heads/")
				current.Branch = branch
			} else if line == "bare" {
				current.IsBare = true
			}
		}
	}

	// Don't forget the last one
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	// Mark the main worktree (first one or bare)
	if len(worktrees) > 0 {
		worktrees[0].IsMain = true
	}

	// Get last commit time for each worktree
	for i := range worktrees {
		if !worktrees[i].IsBare {
			commitTime, _ := cmd.Run(worktrees[i].Path, "log", "-1", "--format=%cr")
			worktrees[i].LastCommit = commitTime
		}
	}

	return worktrees, nil
}

// RemoveWorktree removes a worktree by path.
func RemoveWorktree(repoPath, worktreePath string) error {
	return RemoveWorktreeWithCmd(defaultCmd, repoPath, worktreePath)
}

// RemoveWorktreeWithCmd removes a worktree using a custom commander.
func RemoveWorktreeWithCmd(cmd Commander, repoPath, worktreePath string) error {
	out, err := cmd.Run(repoPath, "worktree", "remove", worktreePath, "--force")
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %v: %s", err, out)
	}
	return nil
}

// PruneWorktrees removes stale worktree references.
func PruneWorktrees(repoPath string) error {
	return PruneWorktreesWithCmd(defaultCmd, repoPath)
}

// PruneWorktreesWithCmd prunes worktrees using a custom commander.
func PruneWorktreesWithCmd(cmd Commander, repoPath string) error {
	out, err := cmd.Run(repoPath, "worktree", "prune")
	if err != nil {
		return fmt.Errorf("failed to prune worktrees: %v: %s", err, out)
	}
	return nil
}

// EnsureRepo ensures a GitHub repo is cloned and up-to-date with the main branch.
// repoSpec is in the format "owner/repo" (e.g., "groq/openbench").
// baseDir is where repos will be stored.
// Returns the path to the local repo directory.
func EnsureRepo(repoSpec, baseDir string) (string, error) {
	return EnsureRepoWithCmd(defaultCmd, repoSpec, baseDir)
}

// EnsureRepoWithCmd ensures a repo is cloned and up-to-date using a custom commander.
func EnsureRepoWithCmd(cmd Commander, repoSpec, baseDir string) (string, error) {
	// Parse repo spec
	parts := strings.Split(repoSpec, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo format: %q (expected 'owner/repo')", repoSpec)
	}
	owner, repo := parts[0], parts[1]

	// Build GitHub URL
	repoURL := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)

	// Create base directory if needed
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create repos directory: %w", err)
	}

	// Local path for this repo
	repoPath := filepath.Join(baseDir, fmt.Sprintf("%s-%s", owner, repo))

	// Check if repo already exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Repo exists, fetch and reset to origin/main
		if err := FetchAndResetWithCmd(cmd, repoPath); err != nil {
			return "", fmt.Errorf("failed to update repo: %w", err)
		}
		return repoPath, nil
	}

	// Clone the repo
	out, err := cmd.Run("", "clone", repoURL, repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to clone repo: %v: %s", err, out)
	}

	return repoPath, nil
}

// FetchAndReset fetches from origin and resets to origin/main.
func FetchAndReset(repoPath string) error {
	return FetchAndResetWithCmd(defaultCmd, repoPath)
}

// FetchAndResetWithCmd fetches and resets using a custom commander.
func FetchAndResetWithCmd(cmd Commander, repoPath string) error {
	// Fetch latest from origin
	out, err := cmd.Run(repoPath, "fetch", "origin", "main")
	if err != nil {
		return fmt.Errorf("fetch failed: %v: %s", err, out)
	}

	// Reset to origin/main
	out, err = cmd.Run(repoPath, "reset", "--hard", "origin/main")
	if err != nil {
		return fmt.Errorf("reset failed: %v: %s", err, out)
	}

	// Clean untracked files
	out, err = cmd.Run(repoPath, "clean", "-fd")
	if err != nil {
		return fmt.Errorf("clean failed: %v: %s", err, out)
	}

	return nil
}

// GetStatusStats returns the number of added and deleted lines in the worktree.
func GetStatusStats(path string) (adds, deletes int, err error) {
	// Use git diff --numstat to get changes compared to HEAD
	out, err := defaultCmd.Run(path, "diff", "HEAD", "--numstat")
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(out, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			var a, d int
			fmt.Sscanf(parts[0], "%d", &a)
			fmt.Sscanf(parts[1], "%d", &d)
			adds += a
			deletes += d
		}
	}
	return adds, deletes, nil
}
