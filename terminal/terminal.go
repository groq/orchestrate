// Package terminal provides iTerm2 terminal management for orchestrate.
package terminal

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"orchestrate/agents"

	"marwan.io/iterm2"
)

// SessionInfo contains information about a terminal session.
// It can represent either an agent worktree or a custom command.
type SessionInfo struct {
	// For agent sessions
	Path   string // Worktree path
	Branch string // Branch name
	Agent  string // Agent name

	// For custom command sessions
	Command        string // Custom command to run (empty = just open terminal)
	Title          string // Custom title
	ColorR         int    // Custom color R (0-255)
	ColorG         int    // Custom color G (0-255)
	ColorB         int    // Custom color B (0-255)
	WorktreePath   string // Path to worktree (commands run here)
	WorktreeBranch string // Branch name (for display in title)

	// Type indicator
	IsCustomCommand bool
}

// WorktreeInfo is an alias for backward compatibility.
type WorktreeInfo = SessionInfo

// Manager handles iTerm2 window and session management.
type Manager struct {
	app          iterm2.App
	MaxPerWindow int
}

// NewManager creates a new terminal manager connected to iTerm2.
func NewManager(appName string) (*Manager, error) {
	app, err := iterm2.NewApp(appName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to iTerm2: %w", err)
	}
	return &Manager{
		app:          app,
		MaxPerWindow: 6,
	}, nil
}

// Close closes the iTerm2 connection.
func (m *Manager) Close() {
	if m.app != nil {
		_ = m.app.Close()
	}
}

// MaximizeWindow maximizes the frontmost iTerm window.
func MaximizeWindow() {
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
	_ = exec.Command("osascript", "-e", script).Run()
}

// BuildSessionCommand builds the shell command to run in a session.
func BuildSessionCommand(s SessionInfo, prompt string) string {
	if s.IsCustomCommand {
		return BuildCustomCommand(s)
	}
	return BuildAgentCommand(s, prompt)
}

// BuildAgentCommand builds the command for an agent session.
func BuildAgentCommand(s SessionInfo, prompt string) string {
	// Set session title: "agent: branch-name"
	title := fmt.Sprintf("%s: %s", s.Agent, s.Branch)
	titleCmd := fmt.Sprintf("echo -ne '\\033]0;%s\\007'", title)

	// Set tab color based on agent (appears in tab bar and pane border)
	colorCmd := ""
	if color, ok := agents.GetColor(s.Agent); ok {
		colorCmd = fmt.Sprintf(" && echo -ne '\\033]6;1;bg;red;brightness;%d\\007\\033]6;1;bg;green;brightness;%d\\007\\033]6;1;bg;blue;brightness;%d\\007'", color.R, color.G, color.B)
	}

	// Build full command
	escapedPrompt := strings.ReplaceAll(prompt, "'", "'\\''")
	return fmt.Sprintf("%s%s && cd \"%s\" && %s '%s'\n", titleCmd, colorCmd, s.Path, s.Agent, escapedPrompt)
}

// BuildCustomCommand builds the command for a custom command session.
func BuildCustomCommand(s SessionInfo) string {
	// Build title - include branch if running in a worktree
	title := s.Title
	if title == "" {
		title = "terminal"
	}
	if s.WorktreeBranch != "" {
		title = fmt.Sprintf("[%s] %s", s.WorktreeBranch, title)
	}
	titleCmd := fmt.Sprintf("echo -ne '\\033]0;%s\\007'", title)

	// Set tab color if specified
	colorCmd := ""
	if s.ColorR > 0 || s.ColorG > 0 || s.ColorB > 0 {
		colorCmd = fmt.Sprintf(" && echo -ne '\\033]6;1;bg;red;brightness;%d\\007\\033]6;1;bg;green;brightness;%d\\007\\033]6;1;bg;blue;brightness;%d\\007'", s.ColorR, s.ColorG, s.ColorB)
	}

	// Change to worktree directory
	cdCmd := ""
	if s.WorktreePath != "" {
		cdCmd = fmt.Sprintf(" && cd \"%s\"", s.WorktreePath)
	}

	// Print branch info if in a worktree
	branchCmd := ""
	if s.WorktreeBranch != "" {
		branchCmd = fmt.Sprintf(" && echo '📍 Branch: %s'", s.WorktreeBranch)
	}

	// Build full command - handle empty commands (just open terminal)
	cmd := strings.TrimSpace(s.Command)
	if cmd == "" || cmd == "\\n" {
		// Empty command - just set title, color, cd, and show branch
		return fmt.Sprintf("%s%s%s%s\n", titleCmd, colorCmd, cdCmd, branchCmd)
	}

	return fmt.Sprintf("%s%s%s%s && %s\n", titleCmd, colorCmd, cdCmd, branchCmd, cmd)
}

// SendCommand sends a command to an iTerm2 session.
func SendCommand(session iterm2.Session, command string) error {
	return session.SendText(command)
}

// CreateGridLayout creates a grid of sessions for a batch of worktrees.
// Returns the sessions in order matching the worktrees.
func CreateGridLayout(sessions []iterm2.Session, count int) ([]iterm2.Session, error) {
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions provided")
	}

	var allSessions []iterm2.Session

	switch count {
	case 1:
		allSessions = []iterm2.Session{sessions[0]}
	case 2:
		// Two columns
		left := sessions[0]
		right, _ := left.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		allSessions = []iterm2.Session{left, right}
	case 3:
		// Three columns
		left := sessions[0]
		middle, _ := left.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		right, _ := middle.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		allSessions = []iterm2.Session{left, middle, right}
	case 4:
		// 2x2 grid
		topLeft := sessions[0]
		topRight, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		bottomLeft, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
		bottomRight, _ := topRight.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
		allSessions = []iterm2.Session{topLeft, topRight, bottomLeft, bottomRight}
	case 5:
		// 3 on top, 2 on bottom
		topLeft := sessions[0]
		topMiddle, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		topRight, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		bottomLeft, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
		bottomRight, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
		allSessions = []iterm2.Session{topLeft, topMiddle, topRight, bottomLeft, bottomRight}
	case 6:
		// 3x2 grid
		topLeft := sessions[0]
		topMiddle, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		topRight, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: true})
		bottomLeft, _ := topLeft.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
		bottomMiddle, _ := topMiddle.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
		bottomRight, _ := topRight.SplitPane(iterm2.SplitPaneOptions{Vertical: false})
		allSessions = []iterm2.Session{topLeft, topMiddle, topRight, bottomLeft, bottomMiddle, bottomRight}
	default:
		// For counts > 6, just return what we have
		allSessions = sessions
	}

	return allSessions, nil
}

// LaunchSessions launches sessions in iTerm2 windows.
func (m *Manager) LaunchSessions(sessions []SessionInfo, prompt string) (int, error) {
	if len(sessions) == 0 {
		return 0, fmt.Errorf("no sessions to launch")
	}

	windowCount := 0

	// Process sessions in batches
	for i := 0; i < len(sessions); i += m.MaxPerWindow {
		// Get batch for this window
		end := i + m.MaxPerWindow
		if end > len(sessions) {
			end = len(sessions)
		}
		batch := sessions[i:end]

		// Create window
		win, err := m.app.CreateWindow()
		if err != nil {
			continue
		}
		MaximizeWindow()
		windowCount++

		tabs, err := win.ListTabs()
		if err != nil || len(tabs) == 0 {
			continue
		}

		itermSessions, err := tabs[0].ListSessions()
		if err != nil || len(itermSessions) == 0 {
			continue
		}

		// Build grid layout
		allSessions, err := CreateGridLayout(itermSessions, len(batch))
		if err != nil {
			continue
		}

		// Send commands to each pane
		for j, session := range allSessions {
			if j < len(batch) {
				cmd := BuildSessionCommand(batch[j], prompt)
				_ = SendCommand(session, cmd)
			}
		}
	}

	return windowCount, nil
}

// LaunchWorktrees launches agents in iTerm2 windows for the given worktrees.
// This is kept for backward compatibility.
func (m *Manager) LaunchWorktrees(worktrees []WorktreeInfo, prompt string) (int, error) {
	return m.LaunchSessions(worktrees, prompt)
}

// WindowCount calculates the number of windows needed for n worktrees.
func (m *Manager) WindowCount(n int) int {
	if n <= 0 {
		return 0
	}
	return (n + m.MaxPerWindow - 1) / m.MaxPerWindow
}
