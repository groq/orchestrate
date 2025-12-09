package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"orchestrate/config"
	"orchestrate/git"
	"orchestrate/terminal"
	"orchestrate/util"
)

func main() {
	// Load config file (if exists)
	result := config.Load("")
	if result.Path != "" {
		fmt.Printf("⚙️  Config: %s\n", result.Path)
	}

	// Get default preset name
	defaultPresetName := ""
	if result.Config != nil {
		defaultPresetName = result.Config.GetDefaultPresetName()
	}

	// Parse CLI flags
	name := flag.String("name", "", "Branch name prefix for worktrees (required)")
	presetFlag := flag.String("preset", defaultPresetName, "Preset name from config file")
	n := flag.Int("n", 0, "Multiplier for agent windows (overrides preset)")
	prompt := flag.String("prompt", "", "Prompt to pass to each agent (required)")
	flag.Parse()

	if *name == "" || *prompt == "" {
		log.Fatal("Error: --name and --prompt are required")
	}

	// Resolve preset settings
	var windows []config.Window
	multiplier := 1

	if result.Config != nil && *presetFlag != "" {
		if preset, ok := result.Config.GetPreset(*presetFlag); ok {
			fmt.Printf("📦 Preset: %s\n", *presetFlag)
			windows = preset
		} else {
			log.Printf("⚠️  Preset '%s' not found, using defaults", *presetFlag)
		}
	}

	// CLI flag sets multiplier
	if *n > 0 {
		multiplier = *n
	}

	// Default to single droid agent if no windows defined
	if len(windows) == 0 {
		windows = []config.Window{{Agent: "droid"}}
	}

	// Get current repo info
	repoRoot, err := git.GetRoot()
	if err != nil {
		log.Fatalf("Error: not in a git repository: %v", err)
	}

	currentBranch, err := git.GetCurrentBranch()
	if err != nil {
		log.Fatalf("Error getting current branch: %v", err)
	}

	fmt.Printf("📂 Repo: %s\n", repoRoot)
	fmt.Printf("🌿 Base branch: %s\n", currentBranch)
	fmt.Printf("💬 Prompt: %s\n", *prompt)

	// Create orchestrator-worktrees directory
	worktreesDir := filepath.Join(filepath.Dir(repoRoot), "orchestrator-worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		log.Fatalf("Failed to create worktrees directory: %v", err)
	}

	// Build all sessions in order
	var sessions []terminal.SessionInfo

	for _, w := range windows {
		// Skip invalid windows (e.g., standalone n: without agent)
		if !w.IsValid() {
			log.Printf("⚠️  Skipping invalid window config (missing agent). Did you mean to put 'n: %d' as a field on an agent?", w.N)
			continue
		}

		// Determine the effective multiplier for this window
		// CLI --n overrides per-window n, otherwise use window's n (defaults to 1)
		effectiveN := w.GetN()
		if multiplier > 1 {
			effectiveN = multiplier
		}

		// For each agent, create worktrees (respecting multiplier)
		for i := 0; i < effectiveN; i++ {
			suffix := util.RandomHex(4)
			branchName := fmt.Sprintf("%s-%s", *name, suffix)
			worktreePath := filepath.Join(worktreesDir, fmt.Sprintf("%s-%s", filepath.Base(repoRoot), branchName))

			err := git.CreateWorktree(repoRoot, worktreePath, branchName, currentBranch)
			if err != nil {
				log.Printf("⚠️  Failed to create worktree for %s: %v", branchName, err)
				continue
			}

			fmt.Printf("✅ Created worktree: %s (branch: %s, agent: %s)\n", worktreePath, branchName, w.Agent)

			// Add the agent session
			sessions = append(sessions, terminal.SessionInfo{
				Path:   worktreePath,
				Branch: branchName,
				Agent:  w.Agent,
			})

			// Add command sessions for this agent's worktree
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
				fmt.Printf("   🖥️  Command: %s (branch: %s)\n", cmd.GetTitle(), branchName)
			}
		}
	}

	if len(sessions) == 0 {
		log.Fatal("No sessions were created successfully")
	}

	// Connect to iTerm2 and launch sessions
	mgr, err := terminal.NewManager("Orchestrate")
	if err != nil {
		log.Fatalf("Failed to connect to iTerm2: %v", err)
	}
	defer mgr.Close()

	windowCount, err := mgr.LaunchSessions(sessions, *prompt)
	if err != nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Printf("\n✨ Started %d session(s) in %d window(s)!\n", len(sessions), windowCount)
}
