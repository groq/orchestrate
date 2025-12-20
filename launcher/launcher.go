// Package launcher provides session launching functionality.
// This is used by both the CLI and TUI to create and launch sessions.
package launcher

import (
	"fmt"
	"os"
	"path/filepath"

	"orchestrate/config"
	"orchestrate/git_utils"
	"orchestrate/terminal"
	"orchestrate/util"
)

// Options contains the options for launching a session.
type Options struct {
	Repo       string
	Name       string
	Prompt     string
	PresetName string
	Multiplier int
	DataDir    string
	Preset     config.Preset
}

// Result contains the result of launching a session.
type Result struct {
	Sessions            []terminal.SessionInfo
	RepoPath            string
	TerminalWindowCount int
	Error               error
}

// Launch creates worktrees and launches sessions.
func Launch(opts Options) Result {
	result := Result{}

	// Validate inputs
	if err := ValidateRepo(opts.Repo); err != nil {
		result.Error = err
		return result
	}
	if err := ValidateName(opts.Name); err != nil {
		result.Error = err
		return result
	}
	if err := ValidatePrompt(opts.Prompt); err != nil {
		result.Error = err
		return result
	}

	// Setup directories
	reposDir := filepath.Join(opts.DataDir, "repos")
	worktreesDir := filepath.Join(opts.DataDir, "worktrees")

	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create worktrees directory: %w", err)
		return result
	}

	// Clone/update repo
	repoRoot, err := git_utils.EnsureRepo(opts.Repo, reposDir)
	if err != nil {
		result.Error = fmt.Errorf("failed to ensure repo: %w", err)
		return result
	}
	result.RepoPath = repoRoot

	// Use main as base branch (consistent with main.go)
	baseBranch := "main"

	// Create sessions from preset
	var sessions []terminal.SessionInfo
	preset := opts.Preset

	if len(preset) == 0 {
		// Default preset with a basic agent
		preset = config.Preset{{Agent: "claude"}}
	}

	for _, w := range preset {
		effectiveN := w.GetN()
		if opts.Multiplier > 1 {
			effectiveN = opts.Multiplier
		}

		for i := 0; i < effectiveN; i++ {
			suffix := util.RandomHex(4)
			branchName := fmt.Sprintf("%s-%s", opts.Name, suffix)
			worktreePath := filepath.Join(worktreesDir, fmt.Sprintf("%s-%s", filepath.Base(repoRoot), branchName))

			err := git_utils.CreateWorktree(repoRoot, worktreePath, branchName, baseBranch)
			if err != nil {
				continue
			}

			// Save session metadata
			sessionMeta := config.CreateSessionMetadata(opts.Repo, branchName, opts.Prompt, opts.PresetName, []string{w.Agent})
			_ = config.SaveSessionMetadata(worktreePath, sessionMeta) // Ignore error - metadata is non-critical

			// Add the agent session
			sessions = append(sessions, terminal.SessionInfo{
				Path:   worktreePath,
				Branch: branchName,
				Agent:  w.Agent,
			})

			// Add command sessions
			for _, cmd := range w.Commands {
				r, g, b, _ := config.ParseHexColor(cmd.Color)
				sessions = append(sessions, terminal.SessionInfo{
					IsCustomCommand: true,
					Command:         cmd.Command,
					Title:           cmd.GetTitle(),
					ColorR:          r,
					ColorG:          g,
					ColorB:          b,
					WorktreePath:    worktreePath,
					WorktreeBranch:  branchName,
				})
			}
		}
	}

	if len(sessions) == 0 {
		result.Error = fmt.Errorf("no sessions were created")
		return result
	}

	result.Sessions = sessions

	// Connect to iTerm2 and launch sessions
	mgr, err := terminal.NewManager("Orchestrate")
	if err != nil {
		result.Error = fmt.Errorf("failed to connect to iTerm2: %w", err)
		return result
	}
	defer mgr.Close()

	windowCount, err := mgr.LaunchSessions(sessions, opts.Prompt)
	if err != nil {
		// Log warning but don't fail
		result.Error = nil
	}

	result.TerminalWindowCount = windowCount
	return result
}

// ValidateRepo checks if a repo string is valid format.
func ValidateRepo(repo string) error {
	if repo == "" {
		return fmt.Errorf("repository is required")
	}
	// Basic validation: should contain a slash
	for i, c := range repo {
		if c == '/' && i > 0 && i < len(repo)-1 {
			return nil
		}
	}
	return fmt.Errorf("invalid repository format, expected 'owner/repo'")
}

// ValidateName checks if a branch name prefix is valid.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name prefix is required")
	}
	// Basic validation: no spaces or special chars
	for _, c := range name {
		if c == ' ' || c == '/' || c == '\\' {
			return fmt.Errorf("branch name cannot contain spaces or slashes")
		}
	}
	return nil
}

// ValidatePrompt checks if a prompt is valid.
func ValidatePrompt(prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	return nil
}
