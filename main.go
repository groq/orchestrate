package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"marwan.io/iterm2"
)

// Preset represents a named configuration preset
type Preset struct {
	N      int      `yaml:"n"`
	Agents []string `yaml:"agents"`
}

// Config represents the .orchestrate.yaml configuration file
type Config struct {
	Default string            `yaml:"default"`
	Presets map[string]Preset `yaml:"presets"`
}

func main() {
	// Load config file (if exists)
	config, configPath := loadConfig()
	if configPath != "" {
		fmt.Printf("⚙️  Config: %s\n", configPath)
	}

	// Get default preset name
	defaultPresetName := ""
	if config != nil && config.Default != "" {
		defaultPresetName = config.Default
	}

	// Parse CLI flags
	name := flag.String("name", "", "Branch name prefix for worktrees (required)")
	presetFlag := flag.String("preset", defaultPresetName, "Preset name from config file")
	n := flag.Int("n", 0, "Multiplier per agent in list (overrides preset)")
	agentsFlag := flag.String("agents", "", "Agents list - repeat for multiples: claude,claude,codex (overrides preset)")
	prompt := flag.String("prompt", "", "Prompt to pass to each agent (required)")
	flag.Parse()

	if *name == "" || *prompt == "" {
		log.Fatal("Error: --name and --prompt are required")
	}

	// Resolve preset settings
	presetN := 1
	presetAgents := "droid"
	if config != nil && *presetFlag != "" {
		if preset, ok := config.Presets[*presetFlag]; ok {
			fmt.Printf("📦 Preset: %s\n", *presetFlag)
			if preset.N > 0 {
				presetN = preset.N
			}
			if len(preset.Agents) > 0 {
				presetAgents = strings.Join(preset.Agents, ",")
			}
		} else {
			log.Printf("⚠️  Preset '%s' not found, using defaults", *presetFlag)
		}
	}

	// CLI flags override preset values
	finalN := presetN
	finalAgents := presetAgents
	if *n > 0 {
		finalN = *n
	}
	if *agentsFlag != "" {
		finalAgents = *agentsFlag
	}

	// Parse agents list
	agents := parseAgents(finalAgents)
	if len(agents) == 0 {
		log.Fatal("Error: at least one valid agent must be specified")
	}

	// Get current repo info
	repoRoot, err := getGitRoot()
	if err != nil {
		log.Fatalf("Error: not in a git repository: %v", err)
	}

	currentBranch, err := getCurrentBranch()
	if err != nil {
		log.Fatalf("Error getting current branch: %v", err)
	}

	fmt.Printf("📂 Repo: %s\n", repoRoot)
	fmt.Printf("🌿 Base branch: %s\n", currentBranch)
	fmt.Printf("🤖 Agents: %v (x%d each)\n", agents, finalN)
	if *prompt != "" {
		fmt.Printf("💬 Prompt: %s\n", *prompt)
	}

	// Create worktrees and collect session info
	type worktreeInfo struct {
		path   string
		branch string
		agent  string
	}
	var worktrees []worktreeInfo

	// Create orchestrator-worktrees directory
	worktreesDir := filepath.Join(filepath.Dir(repoRoot), "orchestrator-worktrees")
	if err := os.MkdirAll(worktreesDir, 0755); err != nil {
		log.Fatalf("Failed to create worktrees directory: %v", err)
	}

	for _, agent := range agents {
		for i := 0; i < finalN; i++ {
			// Generate random hex suffix
			suffix := randomHex(4)
			branchName := fmt.Sprintf("%s-%s", *name, suffix)

			// Create worktree directory path
			worktreePath := filepath.Join(worktreesDir, fmt.Sprintf("%s-%s", filepath.Base(repoRoot), branchName))

			// Create the worktree with new branch
			err := createWorktree(repoRoot, worktreePath, branchName, currentBranch)
			if err != nil {
				log.Printf("⚠️  Failed to create worktree for %s: %v", branchName, err)
				continue
			}

			fmt.Printf("✅ Created worktree: %s (branch: %s)\n", worktreePath, branchName)

			worktrees = append(worktrees, worktreeInfo{
				path:   worktreePath,
				branch: branchName,
				agent:  agent,
			})
		}
	}

	if len(worktrees) == 0 {
		log.Fatal("No worktrees were created successfully")
	}

	// Connect to iTerm2
	app, err := iterm2.NewApp("Orchestrate")
	if err != nil {
		log.Fatalf("Failed to connect to iTerm2: %v", err)
	}
	defer app.Close()

	// Maximize the frontmost iTerm window via AppleScript
	maximizeWindow := func() {
		time.Sleep(100 * time.Millisecond) // Wait for window to be ready
		script := `tell application "iTerm2"
			tell current window
				-- Get screen dimensions
				tell application "Finder"
					set screenBounds to bounds of window of desktop
				end tell
				set screenWidth to item 3 of screenBounds
				set screenHeight to item 4 of screenBounds
				-- Set window to fill most of the screen (accounting for dock/menubar)
				set bounds to {0, 25, screenWidth, screenHeight - 50}
			end tell
		end tell`
		exec.Command("osascript", "-e", script).Run()
	}

	// Agent colors (RGB values for iTerm2 tab color)
	type rgbColor struct{ r, g, b int }
	agentColors := map[string]rgbColor{
		"droid":  {255, 140, 0},   // orange
		"amp":    {220, 50, 50},   // red
		"claude": {210, 180, 140}, // sand/tan
		"codex":  {30, 30, 30},    // black
		"cursor": {65, 105, 225},  // royal blue
	}

	// Helper to send command to a session with title and tab color
	sendCommand := func(session iterm2.Session, wt worktreeInfo) {
		// Set session title: "agent: branch-name"
		title := fmt.Sprintf("%s: %s", wt.agent, wt.branch)
		titleCmd := fmt.Sprintf("echo -ne '\\033]0;%s\\007'", title)

		// Set tab color based on agent (appears in tab bar and pane border)
		colorCmd := ""
		if color, ok := agentColors[wt.agent]; ok {
			colorCmd = fmt.Sprintf(" && echo -ne '\\033]6;1;bg;red;brightness;%d\\007\\033]6;1;bg;green;brightness;%d\\007\\033]6;1;bg;blue;brightness;%d\\007'", color.r, color.g, color.b)
		}

		// Build full command
		escapedPrompt := strings.ReplaceAll(*prompt, "'", "'\\''")
		commands := fmt.Sprintf("%s%s && cd %s && %s '%s'\n", titleCmd, colorCmd, wt.path, wt.agent, escapedPrompt)
		if err := session.SendText(commands); err != nil {
			log.Printf("Failed to send command to %s: %v", wt.branch, err)
		}
		fmt.Printf("🚀 Launched %s in %s\n", wt.agent, wt.branch)
	}

	// Process worktrees in batches of 6 per window
	const maxPerWindow = 6
	for i := 0; i < len(worktrees); i += maxPerWindow {
		// Get batch for this window
		end := i + maxPerWindow
		if end > len(worktrees) {
			end = len(worktrees)
		}
		batch := worktrees[i:end]

		// Create window
		win, err := app.CreateWindow()
		if err != nil {
			log.Printf("Failed to create window: %v", err)
			continue
		}
		maximizeWindow()

		tabs, err := win.ListTabs()
		if err != nil || len(tabs) == 0 {
			log.Printf("No tabs found")
			continue
		}

		sessions, err := tabs[0].ListSessions()
		if err != nil || len(sessions) == 0 {
			log.Printf("No sessions found")
			continue
		}

		// Build grid layout based on batch size
		// Layout: up to 3 columns, 2 rows
		// 1: [1]
		// 2: [1][2]
		// 3: [1][2][3]
		// 4: [1][2]
		//    [3][4]
		// 5: [1][2][3]
		//    [4][5]
		// 6: [1][2][3]
		//    [4][5][6]

		count := len(batch)
		var allSessions []iterm2.Session

		if count == 1 {
			allSessions = []iterm2.Session{sessions[0]}
		} else if count == 2 {
			// Two columns
			left := sessions[0]
			right, _ := left.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			allSessions = []iterm2.Session{left, right}
		} else if count == 3 {
			// Three columns
			left := sessions[0]
			middle, _ := left.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			right, _ := middle.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			allSessions = []iterm2.Session{left, middle, right}
		} else if count == 4 {
			// 2x2 grid
			topLeft := sessions[0]
			topRight, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			bottomLeft, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
			bottomRight, _ := topRight.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
			allSessions = []iterm2.Session{topLeft, topRight, bottomLeft, bottomRight}
		} else if count == 5 {
			// 3 on top, 2 on bottom
			topLeft := sessions[0]
			topMiddle, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			topRight, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			bottomLeft, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
			bottomRight, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
			allSessions = []iterm2.Session{topLeft, topMiddle, topRight, bottomLeft, bottomRight}
		} else if count == 6 {
			// 3x2 grid
			topLeft := sessions[0]
			topMiddle, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			topRight, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
			bottomLeft, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
			bottomMiddle, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
			bottomRight, _ := topRight.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
			allSessions = []iterm2.Session{topLeft, topMiddle, topRight, bottomLeft, bottomMiddle, bottomRight}
		}

		// Send commands to each pane
		for j, session := range allSessions {
			if j < len(batch) {
				sendCommand(session, batch[j])
			}
		}
	}

	fmt.Printf("\n✨ Started %d agent(s) in %d window(s)!\n", len(worktrees), (len(worktrees)+maxPerWindow-1)/maxPerWindow)
}

// loadConfig loads configuration from .orchestrate.yaml in the current directory
func loadConfig() (*Config, string) {
	configFile := ".orchestrate.yaml"

	// Get absolute path for display
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		absPath = configFile
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, "" // No config file, use defaults
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Printf("⚠️  Warning: invalid %s: %v", configFile, err)
		return nil, ""
	}

	return &config, absPath
}

// parseAgents parses the comma-separated agents string
func parseAgents(s string) []string {
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, ",")

	var agents []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			agents = append(agents, p)
		}
	}
	return agents
}

// randomHex generates a random hex string of n bytes
func randomHex(n int) string {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// getGitRoot returns the root directory of the current git repository
func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// getCurrentBranch returns the current git branch name
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// createWorktree creates a new git worktree with a new branch
func createWorktree(repoRoot, worktreePath, branchName, baseBranch string) error {
	// Check if worktree path already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree path already exists: %s", worktreePath)
	}

	// Create worktree with new branch based on current branch
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath, baseBranch)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}
