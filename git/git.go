// Package git provides git-related operations for orchestrate.
package git

import (
	"fmt"
	"os"
	"os/exec"
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
