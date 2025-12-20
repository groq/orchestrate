package terminal

import (
	"strings"
	"testing"
)

// TestBuildAgentCommand_EmptyAgent documents the behavior when agent is empty.
// This case is now caught upstream by config.Window.IsValid() in main.go,
// so it should never reach BuildAgentCommand in practice.
// This test ensures we understand and document the behavior.
func TestBuildAgentCommand_EmptyAgent(t *testing.T) {
	session := SessionInfo{
		Path:   "/path/to/worktree",
		Branch: "test-branch",
		Agent:  "", // Empty agent - should be caught by IsValid() upstream
	}

	cmd := BuildAgentCommand(session, "Fix the bug")

	// Document: empty agent produces these patterns (caught upstream now)
	// - Title: ": branch" (empty before colon)
	// - Command: "&&  'prompt'" (double space, no agent)
	if !strings.Contains(cmd, "\\033]0;: ") {
		t.Logf("Expected title pattern with empty agent, got: %s", cmd)
	}

	// The important thing is the command ends properly (doesn't crash)
	if !strings.HasSuffix(cmd, "\n") {
		t.Errorf("Command should end with newline, got: %s", cmd)
	}
}

func TestBuildAgentCommand(t *testing.T) {
	tests := []struct {
		name      string
		session   SessionInfo
		prompt    string
		wantTitle string
		wantAgent string
		wantPath  string
		wantColor bool
	}{
		{
			name: "basic agent command",
			session: SessionInfo{
				Path:   "/path/to/worktree",
				Branch: "feature-abc123",
				Agent:  "claude",
			},
			prompt:    "Fix the bug",
			wantTitle: "claude: feature-abc123",
			wantAgent: "claude",
			wantPath:  "/path/to/worktree",
			wantColor: true,
		},
		{
			name: "unknown agent - no color",
			session: SessionInfo{
				Path:   "/path/to/worktree",
				Branch: "test-branch",
				Agent:  "unknown-agent",
			},
			prompt:    "Do something",
			wantTitle: "unknown-agent: test-branch",
			wantAgent: "unknown-agent",
			wantPath:  "/path/to/worktree",
			wantColor: false,
		},
		{
			name: "prompt with single quotes",
			session: SessionInfo{
				Path:   "/path/to/worktree",
				Branch: "branch",
				Agent:  "droid",
			},
			prompt:    "Fix the user's bug",
			wantTitle: "droid: branch",
			wantAgent: "droid",
			wantPath:  "/path/to/worktree",
			wantColor: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildAgentCommand(tt.session, tt.prompt)

			if !strings.Contains(cmd, tt.wantTitle) {
				t.Errorf("Command should contain title %q, got: %s", tt.wantTitle, cmd)
			}

			if !strings.Contains(cmd, tt.session.Agent) {
				t.Errorf("Command should contain agent %q, got: %s", tt.session.Agent, cmd)
			}

			if !strings.Contains(cmd, tt.wantPath) {
				t.Errorf("Command should contain path %q, got: %s", tt.wantPath, cmd)
			}

			if tt.wantColor {
				if !strings.Contains(cmd, "brightness") {
					t.Errorf("Command should contain color codes for known agent, got: %s", cmd)
				}
			} else {
				if strings.Contains(cmd, "brightness") {
					t.Errorf("Command should NOT contain color codes for unknown agent, got: %s", cmd)
				}
			}

			if !strings.HasSuffix(cmd, "\n") {
				t.Errorf("Command should end with newline, got: %s", cmd)
			}
		})
	}
}

func TestBuildCustomCommand_Basic(t *testing.T) {
	tests := []struct {
		name      string
		session   SessionInfo
		wantTitle string
		wantCmd   string
		wantColor bool
	}{
		{
			name: "basic custom command with worktree",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "npm run dev",
				Title:           "Dev Server",
				WorktreePath:    "/path/to/worktree",
				WorktreeBranch:  "fix-bug-123",
			},
			wantTitle: "[fix-bug-123] Dev Server",
			wantCmd:   "npm run dev",
			wantColor: false,
		},
		{
			name: "custom command with color",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "npm test",
				Title:           "Tests",
				ColorR:          255,
				ColorG:          255,
				ColorB:          0,
				WorktreePath:    "/path",
				WorktreeBranch:  "branch",
			},
			wantTitle: "[branch] Tests",
			wantCmd:   "npm test",
			wantColor: true,
		},
		{
			name: "custom command without title",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "echo hello",
				WorktreePath:    "/path",
				WorktreeBranch:  "branch",
			},
			wantTitle: "[branch] terminal",
			wantCmd:   "echo hello",
			wantColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildCustomCommand(tt.session)

			if !strings.Contains(cmd, tt.wantTitle) {
				t.Errorf("Command should contain title %q, got: %s", tt.wantTitle, cmd)
			}

			if !strings.Contains(cmd, tt.wantCmd) {
				t.Errorf("Command should contain command %q, got: %s", tt.wantCmd, cmd)
			}

			if tt.wantColor {
				if !strings.Contains(cmd, "brightness") {
					t.Errorf("Command should contain color codes, got: %s", cmd)
				}
			}

			if !strings.HasSuffix(cmd, "\n") {
				t.Errorf("Command should end with newline, got: %s", cmd)
			}
		})
	}
}

func TestBuildCustomCommand_CdToWorktree(t *testing.T) {
	tests := []struct {
		name          string
		session       SessionInfo
		wantCd        bool
		wantCdPath    string
		wantBranchMsg bool
	}{
		{
			name: "with worktree path and branch",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "npm run dev",
				Title:           "Dev",
				WorktreePath:    "/home/user/project-fix-123",
				WorktreeBranch:  "fix-123",
			},
			wantCd:        true,
			wantCdPath:    "/home/user/project-fix-123",
			wantBranchMsg: true,
		},
		{
			name: "without worktree",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "npm run dev",
				Title:           "Dev",
			},
			wantCd:        false,
			wantBranchMsg: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildCustomCommand(tt.session)

			if tt.wantCd {
				if !strings.Contains(cmd, "cd \""+tt.wantCdPath+"\"") {
					t.Errorf("Should cd to worktree, got: %s", cmd)
				}
			} else {
				if strings.Contains(cmd, "cd ") {
					t.Errorf("Should NOT cd, got: %s", cmd)
				}
			}

			if tt.wantBranchMsg {
				if !strings.Contains(cmd, "Branch:") {
					t.Errorf("Should show branch message, got: %s", cmd)
				}
			} else {
				if strings.Contains(cmd, "Branch:") {
					t.Errorf("Should NOT show branch message, got: %s", cmd)
				}
			}
		})
	}
}

func TestBuildCustomCommand_EmptyCommand(t *testing.T) {
	tests := []struct {
		name    string
		session SessionInfo
	}{
		{
			name: "empty command string",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "",
				Title:           "Terminal",
				WorktreePath:    "/path",
				WorktreeBranch:  "branch",
			},
		},
		{
			name: "whitespace only command",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "   ",
				Title:           "Terminal",
				WorktreePath:    "/path",
				WorktreeBranch:  "branch",
			},
		},
		{
			name: "backslash-n command",
			session: SessionInfo{
				IsCustomCommand: true,
				Command:         "\\n",
				Title:           "Terminal",
				WorktreePath:    "/path",
				WorktreeBranch:  "branch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildCustomCommand(tt.session)

			// Should still set title with branch
			if !strings.Contains(cmd, "[branch] Terminal") {
				t.Errorf("Should have title with branch, got: %s", cmd)
			}

			// Should cd to worktree
			if !strings.Contains(cmd, "cd \"/path\"") {
				t.Errorf("Should cd to worktree, got: %s", cmd)
			}

			// Should show branch message
			if !strings.Contains(cmd, "Branch: branch") {
				t.Errorf("Should show branch message, got: %s", cmd)
			}

			// Should end with newline
			if !strings.HasSuffix(cmd, "\n") {
				t.Errorf("Should end with newline, got: %s", cmd)
			}

			// Should NOT have the actual empty command executed
			// (no " && " followed by empty/whitespace at the end)
			if strings.HasSuffix(strings.TrimSpace(cmd), "&& ") {
				t.Errorf("Should not have trailing &&, got: %s", cmd)
			}
		})
	}
}

func TestBuildCustomCommand_ColorVariations(t *testing.T) {
	tests := []struct {
		name      string
		r, g, b   int
		wantColor bool
	}{
		{"no color (all zeros)", 0, 0, 0, false},
		{"red only", 255, 0, 0, true},
		{"green only", 0, 255, 0, true},
		{"blue only", 0, 0, 255, true},
		{"full color", 255, 128, 64, true},
		{"low values", 1, 1, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := SessionInfo{
				IsCustomCommand: true,
				Command:         "test",
				Title:           "Test",
				ColorR:          tt.r,
				ColorG:          tt.g,
				ColorB:          tt.b,
				WorktreePath:    "/path",
				WorktreeBranch:  "branch",
			}
			cmd := BuildCustomCommand(session)

			if tt.wantColor {
				if !strings.Contains(cmd, "brightness") {
					t.Errorf("Should contain color codes, got: %s", cmd)
				}
			} else {
				if strings.Contains(cmd, "brightness") {
					t.Errorf("Should NOT contain color codes, got: %s", cmd)
				}
			}
		})
	}
}

func TestBuildSessionCommand_Dispatch(t *testing.T) {
	t.Run("dispatches to agent", func(t *testing.T) {
		s := SessionInfo{
			Path:   "/path",
			Branch: "branch",
			Agent:  "claude",
		}
		cmd := BuildSessionCommand(s, "test prompt")
		if !strings.Contains(cmd, "claude") {
			t.Error("Should dispatch to agent command")
		}
	})

	t.Run("dispatches to custom", func(t *testing.T) {
		s := SessionInfo{
			IsCustomCommand: true,
			Command:         "npm run dev",
			Title:           "Dev",
			WorktreePath:    "/path",
			WorktreeBranch:  "branch",
		}
		cmd := BuildSessionCommand(s, "ignored")
		if !strings.Contains(cmd, "npm run dev") {
			t.Error("Should dispatch to custom command")
		}
	})
}

func TestSessionInfo_Fields(t *testing.T) {
	// Agent session
	agent := SessionInfo{
		Path:   "/test/path",
		Branch: "test-branch",
		Agent:  "claude",
	}

	if agent.Path != "/test/path" {
		t.Errorf("Path = %q, want '/test/path'", agent.Path)
	}
	if agent.Branch != "test-branch" {
		t.Errorf("Branch = %q, want 'test-branch'", agent.Branch)
	}
	if agent.Agent != "claude" {
		t.Errorf("Agent = %q, want 'claude'", agent.Agent)
	}
	if agent.IsCustomCommand {
		t.Error("IsCustomCommand should be false for agent")
	}

	// Custom command session
	custom := SessionInfo{
		IsCustomCommand: true,
		Command:         "npm run dev",
		Title:           "Dev Server",
		ColorR:          255,
		ColorG:          128,
		ColorB:          0,
		WorktreePath:    "/worktree/path",
		WorktreeBranch:  "feature-branch",
	}

	if !custom.IsCustomCommand {
		t.Error("IsCustomCommand should be true")
	}
	if custom.Command != "npm run dev" {
		t.Errorf("Command = %q, want 'npm run dev'", custom.Command)
	}
	if custom.Title != "Dev Server" {
		t.Errorf("Title = %q, want 'Dev Server'", custom.Title)
	}
	if custom.WorktreePath != "/worktree/path" {
		t.Errorf("WorktreePath = %q, want '/worktree/path'", custom.WorktreePath)
	}
	if custom.WorktreeBranch != "feature-branch" {
		t.Errorf("WorktreeBranch = %q, want 'feature-branch'", custom.WorktreeBranch)
	}
}

func TestManager_TerminalWindowCount(t *testing.T) {
	tests := []struct {
		name         string
		maxPerWindow int
		n            int
		want         int
	}{
		{"6 sessions, max 6 per window", 6, 6, 1},
		{"7 sessions, max 6 per window", 6, 7, 2},
		{"12 sessions, max 6 per window", 6, 12, 2},
		{"13 sessions, max 6 per window", 6, 13, 3},
		{"1 session", 6, 1, 1},
		{"0 sessions", 6, 0, 0},
		{"negative sessions", 6, -1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{MaxPerWindow: tt.maxPerWindow}
			got := m.TerminalWindowCount(tt.n)
			if got != tt.want {
				t.Errorf("TerminalWindowCount(%d) = %d, want %d", tt.n, got, tt.want)
			}
		})
	}
}

func TestBuildAgentCommand_EscapeQuotes(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   string
	}{
		{"no quotes", "simple prompt", "simple prompt"},
		{"single quote", "it's working", "it'\\''s working"},
		{"multiple single quotes", "it's working and it's great", "it'\\''s working and it'\\''s great"},
		{"double quotes - not escaped", `say "hello"`, `say "hello"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := SessionInfo{Path: "/path", Branch: "branch", Agent: "droid"}
			cmd := BuildAgentCommand(s, tt.prompt)

			if !strings.Contains(cmd, tt.want) {
				t.Errorf("Command should contain escaped prompt %q, got: %s", tt.want, cmd)
			}
		})
	}
}

func TestBuildAgentCommand_AllAgentColors(t *testing.T) {
	knownAgents := []string{"droid", "claude", "codex"}

	for _, agent := range knownAgents {
		t.Run(agent, func(t *testing.T) {
			s := SessionInfo{
				Path:   "/path",
				Branch: "branch",
				Agent:  agent,
			}
			cmd := BuildAgentCommand(s, "test")

			if !strings.Contains(cmd, "brightness") {
				t.Errorf("Agent %q should have color codes, but command was: %s", agent, cmd)
			}
		})
	}
}

func TestBuildCustomCommand_FullFeatured(t *testing.T) {
	s := SessionInfo{
		IsCustomCommand: true,
		Command:         "npm run dev -- --port 3000",
		Title:           "Dev Server",
		ColorR:          0,
		ColorG:          255,
		ColorB:          0,
		WorktreePath:    "/home/user/project-fix-abc123",
		WorktreeBranch:  "fix-abc123",
	}
	cmd := BuildCustomCommand(s)

	// Check title with branch
	if !strings.Contains(cmd, "[fix-abc123] Dev Server") {
		t.Errorf("Should have branch and title, got: %s", cmd)
	}

	// Check color
	if !strings.Contains(cmd, "brightness;0") || !strings.Contains(cmd, "brightness;255") {
		t.Errorf("Should have color codes, got: %s", cmd)
	}

	// Check cd
	if !strings.Contains(cmd, "cd \"/home/user/project-fix-abc123\"") {
		t.Errorf("Should cd to worktree, got: %s", cmd)
	}

	// Check branch message
	if !strings.Contains(cmd, "Branch: fix-abc123") {
		t.Errorf("Should have branch message, got: %s", cmd)
	}

	// Check command
	if !strings.Contains(cmd, "npm run dev -- --port 3000") {
		t.Errorf("Should have command, got: %s", cmd)
	}
}

func TestBuildCustomCommand_OrderOfOperations(t *testing.T) {
	s := SessionInfo{
		IsCustomCommand: true,
		Command:         "my-command",
		Title:           "My Title",
		ColorR:          100,
		ColorG:          100,
		ColorB:          100,
		WorktreePath:    "/path",
		WorktreeBranch:  "branch",
	}
	cmd := BuildCustomCommand(s)

	// Find positions of key elements
	titlePos := strings.Index(cmd, "My Title")
	colorPos := strings.Index(cmd, "brightness")
	cdPos := strings.Index(cmd, "cd \"/path\"")
	branchPos := strings.Index(cmd, "Branch:")
	cmdPos := strings.LastIndex(cmd, "my-command")

	if titlePos == -1 || colorPos == -1 || cdPos == -1 || branchPos == -1 || cmdPos == -1 {
		t.Fatalf("Missing expected elements in command: %s", cmd)
	}

	// Verify order: title < color < cd < branch < command
	if titlePos >= colorPos || colorPos >= cdPos || cdPos >= branchPos || branchPos >= cmdPos {
		t.Errorf("Command elements not in expected order. title=%d, color=%d, cd=%d, branch=%d, cmd=%d. Full: %s",
			titlePos, colorPos, cdPos, branchPos, cmdPos, cmd)
	}
}

func TestManager_Fields(t *testing.T) {
	m := &Manager{MaxPerWindow: 4}

	if m.MaxPerWindow != 4 {
		t.Errorf("Manager.MaxPerWindow = %d, want 4", m.MaxPerWindow)
	}
}
