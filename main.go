package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"orchestrate/config"
	"orchestrate/git_utils"
	tui "orchestrate/internal/tui"
	"orchestrate/terminal"
	"orchestrate/util"
)

func main() {
	// Set up data directory (platform-appropriate location)
	dataDir, err := util.DataDir()
	if err != nil {
		log.Fatalf("Failed to get data directory: %v", err)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Parse CLI flags first to check for UI mode
	uiMode := flag.Bool("ui", false, "Launch the interactive TUI")
	repo := flag.String("repo", "", "GitHub repo to clone (e.g., 'groq/openbench') (required for CLI mode)")
	name := flag.String("name", "", "Branch name prefix for worktrees (required for CLI mode)")
	presetFlag := flag.String("preset", "", "Preset name from config file")
	n := flag.Int("n", 0, "Multiplier for agent worktrees (overrides preset)")
	prompt := flag.String("prompt", "", "Prompt to pass to each agent (required for CLI mode)")
	flag.Parse()

	// Load app settings (from orchestrate.yaml)
	appSettingsPath := filepath.Join(dataDir, config.AppSettingsFileName)
	appSettings, _, err := config.LoadAppSettings(dataDir)
	if err != nil {
		log.Printf("Warning: Could not load app settings: %v", err)
		appSettings = config.DefaultAppSettings()
	}

	// Create orchestrate.yaml if it doesn't exist
	if _, err := os.Stat(appSettingsPath); os.IsNotExist(err) {
		if err := config.SaveAppSettings(dataDir, appSettings); err != nil {
			log.Printf("Warning: Could not save default app settings: %v", err)
		}
	}

	// Load preset config file from data directory (settings.yaml)
	settingsPath := filepath.Join(dataDir, config.SettingsFileName)
	result := config.Load(dataDir)

	// If UI mode, launch the TUI
	if *uiMode {
		if err := tui.Run(dataDir, appSettings, result.Config); err != nil {
			log.Fatalf("Error running TUI: %v", err)
		}
		return
	}

	// CLI mode - require settings file and CLI args
	if result.Config == nil {
		displayPath := util.DisplayPath(settingsPath)
		log.Fatalf("Error: Settings file not found.\n\nPlease create: %s\n\nExample:\n\ndefault: default\n\npresets:\n  default:\n    - agent: claude\n", displayPath)
	}
	fmt.Printf("Settings: %s\n", result.Path)
	fmt.Printf("App Settings: %s\n", appSettingsPath)

	// Get default preset name
	defaultPresetName := result.Config.GetDefaultPresetName()

	// Use app settings default if config doesn't have one
	if defaultPresetName == "" && appSettings.Session.DefaultPreset != "" {
		defaultPresetName = appSettings.Session.DefaultPreset
	}

	// Apply preset flag default if not set
	if *presetFlag == "" {
		*presetFlag = defaultPresetName
	}

	if *repo == "" || *name == "" || *prompt == "" {
		fmt.Println("Usage: orchestrate [options]")
		fmt.Println()
		fmt.Println("Modes:")
		fmt.Println("  --ui              Launch the interactive TUI to manage settings")
		fmt.Println()
		fmt.Println("CLI Mode (requires --repo, --name, --prompt):")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  orchestrate --ui                                    # Launch TUI")
		fmt.Println("  orchestrate --repo owner/repo --name feature --prompt 'Fix bug'")
		os.Exit(1)
	}

	// Resolve preset settings
	var worktrees []config.Worktree
	multiplier := 1

	if result.Config != nil && *presetFlag != "" {
		if preset, ok := result.Config.GetPreset(*presetFlag); ok {
			fmt.Printf("Preset: %s\n", *presetFlag)
			worktrees = preset
		} else {
			log.Printf("Warning: Preset '%s' not found", *presetFlag)
		}
	}

	// CLI flag sets multiplier
	if *n > 0 {
		multiplier = *n
	}

	// Default to single droid agent if no worktrees defined
	if len(worktrees) == 0 {
		worktrees = []config.Worktree{{Agent: "droid"}}
	}

	reposDir := filepath.Join(dataDir, "repos")

	// Clone or update the repo from main branch
	fmt.Printf("Repo: %s\n", *repo)
	fmt.Printf("Fetching latest from main branch...\n")

	repoRoot, err := git_utils.EnsureRepo(*repo, reposDir)
	if err != nil {
		log.Fatalf("Error: failed to ensure repo: %v", err)
	}

	baseBranch := "main"

	fmt.Printf("Local path: %s\n", repoRoot)
	fmt.Printf("Base branch: %s\n", baseBranch)
	fmt.Printf("Prompt: %s\n", *prompt)

	// Create worktrees directory in the data directory
	worktreesDir := filepath.Join(dataDir, "worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		log.Fatalf("Failed to create worktrees directory: %v", err)
	}

	// Build all sessions in order
	var sessions []terminal.SessionInfo

	for _, w := range worktrees {
		// Skip invalid worktrees (e.g., standalone n: without agent)
		if !w.IsValid() {
			log.Printf("Warning: Skipping invalid worktree config")
			continue
		}

		// Determine the effective multiplier for this worktree
		// CLI --n overrides per-worktree n, otherwise use worktree's n (defaults to 1)
		effectiveN := w.GetN()
		if multiplier > 1 {
			effectiveN = multiplier
		}

		// For each agent, create worktrees (respecting multiplier)
		for i := 0; i < effectiveN; i++ {
			suffix := util.RandomHex(4)
			branchName := fmt.Sprintf("%s-%s", *name, suffix)
			worktreePath := filepath.Join(worktreesDir, fmt.Sprintf("%s-%s", filepath.Base(repoRoot), branchName))

			err := git_utils.CreateWorktree(repoRoot, worktreePath, branchName, baseBranch)
			if err != nil {
				log.Printf("Warning: Failed to create worktree for %s", branchName)
				continue
			}

			// Save session metadata for this worktree
			sessionMeta := config.CreateSessionMetadata(*repo, branchName, *prompt, *presetFlag, []string{w.Agent})
			if err := config.SaveSessionMetadata(worktreePath, sessionMeta); err != nil {
				log.Printf("Warning: Failed to save session metadata")
			}

			fmt.Printf("Created worktree: %s", worktreePath)

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
				fmt.Printf("   Command: %s (branch: %s)\n", cmd.GetTitle(), branchName)
			}
		}
	}

	if len(sessions) == 0 {
		log.Fatal("No sessions were created successfully")
	}

	// Use terminal type from app settings
	if appSettings.Terminal.Type == config.TerminalRegular {
		fmt.Printf("\nWarning: Regular terminal mode not yet implemented")
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

	fmt.Printf("\nStarted %d session(s) in %d worktree(s)!\n", len(sessions), windowCount)
}
