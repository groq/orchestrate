package worktrees

import (
	"orchestrate/config"
	"orchestrate/git_utils"
	"orchestrate/internal/tui/context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func createMockAppSettings() *config.AppSettings {
	return &config.AppSettings{
		UI: config.UISettings{
			Theme: "default",
		},
	}
}

func TestNew(t *testing.T) {
	appSettings := createMockAppSettings()
	ctx := context.NewProgramContext("/test/data", appSettings, nil)
	m := New(ctx)

	if m.ctx != ctx {
		t.Error("Context not set correctly")
	}
	if m.loading {
		t.Error("Should not be loading initially")
	}
	if m.selected != 0 {
		t.Error("Selected should start at 0")
	}
	if !strings.HasSuffix(m.worktreesDir, "worktrees") {
		t.Errorf("Worktrees dir not set correctly: %s", m.worktreesDir)
	}
}

func TestIsAtTop(t *testing.T) {
	m := Model{
		selected: 0,
		worktrees: []WorktreeItem{
			{Name: "wt1"},
			{Name: "wt2"},
		},
	}

	if !m.IsAtTop() {
		t.Error("Should be at top when selected = 0")
	}

	m.selected = 1
	if m.IsAtTop() {
		t.Error("Should not be at top when selected > 0")
	}
}

func TestSelectedWorktree(t *testing.T) {
	m := Model{
		selected: 1,
		worktrees: []WorktreeItem{
			{Name: "wt1"},
			{Name: "wt2"},
			{Name: "wt3"},
		},
	}

	wt := m.SelectedWorktree()
	if wt == nil {
		t.Fatal("SelectedWorktree should not be nil")
	}
	if wt.Name != "wt2" {
		t.Errorf("Expected wt2, got %s", wt.Name)
	}

	// Test out of bounds
	m.selected = 10
	wt = m.SelectedWorktree()
	if wt != nil {
		t.Error("Should return nil for out of bounds")
	}

	// Test empty worktrees
	m.worktrees = nil
	m.selected = 0
	wt = m.SelectedWorktree()
	if wt != nil {
		t.Error("Should return nil for empty worktrees")
	}
}

func TestUpdateNavigation(t *testing.T) {
	m := Model{
		selected: 1,
		worktrees: []WorktreeItem{
			{Name: "wt1"},
			{Name: "wt2"},
			{Name: "wt3"},
		},
	}

	// Test down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.selected != 2 {
		t.Errorf("Down should move to 2, got %d", m.selected)
	}

	// Test down at end (should not move)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.selected != 2 {
		t.Errorf("Should stay at 2, got %d", m.selected)
	}

	// Test up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.selected != 1 {
		t.Errorf("Up should move to 1, got %d", m.selected)
	}

	// Test g (go to top)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.selected != 0 {
		t.Errorf("g should go to top, got %d", m.selected)
	}

	// Test G (go to bottom)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.selected != 2 {
		t.Errorf("G should go to bottom, got %d", m.selected)
	}
}

func TestUpdateEnterKey(t *testing.T) {
	m := Model{
		selected: 0,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
		},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter key should return a command")
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("Command should return a message")
	}

	focusMsg, ok := msg.(FocusWorktreeMsg)
	if !ok {
		t.Fatalf("Expected FocusWorktreeMsg, got %T", msg)
	}

	if focusMsg.Worktree == nil {
		t.Error("Worktree should not be nil")
	}
	if focusMsg.Worktree.Name != "wt1" {
		t.Errorf("Expected wt1, got %s", focusMsg.Worktree.Name)
	}
}

func TestUpdateDetailsKey(t *testing.T) {
	m := Model{
		selected: 0,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
		},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("'d' key should return a command")
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("Command should return a message")
	}

	detailsMsg, ok := msg.(WorktreeDetailsMsg)
	if !ok {
		t.Fatalf("Expected WorktreeDetailsMsg, got %T", msg)
	}

	if detailsMsg.Worktree == nil {
		t.Error("Worktree should not be nil")
	}
	if detailsMsg.Worktree.Name != "wt1" {
		t.Errorf("Expected wt1, got %s", detailsMsg.Worktree.Name)
	}
}

func TestUpdateOpenKey(t *testing.T) {
	m := Model{
		selected: 0,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1", HasMeta: true},
		},
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if cmd == nil {
		t.Fatal("'o' key should return a command")
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("Command should return a message")
	}

	openMsg, ok := msg.(OpenWorktreeMsg)
	if !ok {
		t.Fatalf("Expected OpenWorktreeMsg, got %T", msg)
	}

	if openMsg.Worktree == nil {
		t.Error("Worktree should not be nil")
	}
	if openMsg.Worktree.Name != "wt1" {
		t.Errorf("Expected wt1, got %s", openMsg.Worktree.Name)
	}
}

func TestWorktreesLoadedMsg(t *testing.T) {
	m := Model{
		loading: true,
		worktrees: []WorktreeItem{
			{Name: "old"},
		},
	}

	newWorktrees := []WorktreeItem{
		{Name: "wt1"},
		{Name: "wt2"},
	}

	msg := WorktreesLoadedMsg{
		Worktrees: newWorktrees,
		Err:       nil,
	}

	m, _ = m.Update(msg)

	if m.loading {
		t.Error("Should not be loading after WorktreesLoadedMsg")
	}
	if len(m.worktrees) != 2 {
		t.Errorf("Expected 2 worktrees, got %d", len(m.worktrees))
	}
	if m.worktrees[0].Name != "wt1" {
		t.Errorf("Expected wt1, got %s", m.worktrees[0].Name)
	}
}

func TestRenderActionsContent(t *testing.T) {
	appSettings := createMockAppSettings()
	ctx := context.NewProgramContext("/test/data", appSettings, nil)
	m := Model{
		ctx: ctx,
		worktrees: []WorktreeItem{
			{Name: "wt1"},
		},
	}

	actions := m.renderActions()
	if actions == "" {
		t.Error("Actions should not be empty")
	}

	// Check for key action labels
	expectedKeys := []string{"navigate", "focus", "open", "details", "delete", "refresh"}
	for _, key := range expectedKeys {
		if !strings.Contains(actions, key) {
			t.Errorf("Actions should contain '%s'", key)
		}
	}
}

func TestSetDimensions(t *testing.T) {
	m := Model{}
	dims := struct {
		Width  int
		Height int
	}{
		Width:  100,
		Height: 50,
	}

	// Convert to constants.Dimensions
	constDims := struct{ Width, Height int }{dims.Width, dims.Height}

	// Use reflection to convert or just create directly
	m.dimensions.Width = dims.Width
	m.dimensions.Height = dims.Height

	if m.dimensions.Width != 100 {
		t.Errorf("Width not set correctly: %d", m.dimensions.Width)
	}
	if m.dimensions.Height != 50 {
		t.Errorf("Height not set correctly: %d", m.dimensions.Height)
	}

	// Test with actual SetDimensions method
	m = Model{}
	m.SetDimensions(constDims)
	if m.dimensions.Width != 100 {
		t.Errorf("Width not set correctly via SetDimensions: %d", m.dimensions.Width)
	}
	if m.dimensions.Height != 50 {
		t.Errorf("Height not set correctly via SetDimensions: %d", m.dimensions.Height)
	}
}

func TestUpdateProgramContext(t *testing.T) {
	m := Model{}
	appSettings := createMockAppSettings()
	ctx := context.NewProgramContext("/new/data", appSettings, nil)

	m.UpdateProgramContext(ctx)

	if m.ctx != ctx {
		t.Error("Context not updated")
	}
	if !strings.HasSuffix(m.worktreesDir, "worktrees") {
		t.Error("Worktrees dir not updated")
	}
}

func TestWorktreeItemWithFileStats(t *testing.T) {
	item := WorktreeItem{
		Name:    "test-worktree",
		Path:    "/test/path",
		Branch:  "feature-123",
		Repo:    "owner/repo",
		Adds:    10,
		Deletes: 5,
		FileStats: []git_utils.FileStats{
			{Path: "main.go", Adds: 5, Deletes: 2},
			{Path: "util.go", Adds: 5, Deletes: 3},
		},
		HasMeta:    true,
		PresetName: "default",
		Agents:     []string{"claude"},
		Prompt:     "Fix bug",
		CreatedAt:  time.Now(),
	}

	if len(item.FileStats) != 2 {
		t.Errorf("Expected 2 file stats, got %d", len(item.FileStats))
	}
	if item.FileStats[0].Path != "main.go" {
		t.Errorf("Expected main.go, got %s", item.FileStats[0].Path)
	}
	if item.Adds != 10 {
		t.Errorf("Expected 10 adds, got %d", item.Adds)
	}
}

func TestMinFunction(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{0, 0, 0},
		{-1, 5, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestDetailsToggleWithD(t *testing.T) {
	m := Model{
		selected:    0,
		detailsMode: false,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
		},
	}

	// First 'd' press should enable details mode and send WorktreeDetailsMsg
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !m.detailsMode {
		t.Error("Details mode should be enabled after first 'd' press")
	}
	if cmd == nil {
		t.Fatal("First 'd' press should return a command")
	}
	msg := cmd()
	if _, ok := msg.(WorktreeDetailsMsg); !ok {
		t.Errorf("Expected WorktreeDetailsMsg, got %T", msg)
	}

	// Second 'd' press should disable details mode and send CloseWorktreeDetailsMsg
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.detailsMode {
		t.Error("Details mode should be disabled after second 'd' press")
	}
	if cmd == nil {
		t.Fatal("Second 'd' press should return a command")
	}
	msg = cmd()
	if _, ok := msg.(CloseWorktreeDetailsMsg); !ok {
		t.Errorf("Expected CloseWorktreeDetailsMsg, got %T", msg)
	}
}

func TestEscClosesDetails(t *testing.T) {
	m := Model{
		selected:    0,
		detailsMode: true,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
		},
	}

	// Esc should close details mode
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.detailsMode {
		t.Error("Details mode should be disabled after esc press")
	}
	if cmd == nil {
		t.Fatal("Esc press should return a command when in details mode")
	}
	msg := cmd()
	if _, ok := msg.(CloseWorktreeDetailsMsg); !ok {
		t.Errorf("Expected CloseWorktreeDetailsMsg, got %T", msg)
	}
}

func TestEscWhenNotInDetailsMode(t *testing.T) {
	m := Model{
		selected:    0,
		detailsMode: false,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
		},
	}

	// Esc when not in details mode should not send a message
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.detailsMode {
		t.Error("Details mode should still be disabled")
	}
	// We just need to ensure detailsMode is still false
}

func TestNavigationUpdatesDetailsWhenInDetailsMode(t *testing.T) {
	m := Model{
		selected:    0,
		detailsMode: true,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
			{Name: "wt2", Path: "/test/wt2"},
			{Name: "wt3", Path: "/test/wt3"},
		},
	}

	// Navigate down - should send WorktreeDetailsMsg for new selection
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.selected != 1 {
		t.Errorf("Expected selection to move to 1, got %d", m.selected)
	}
	if cmd == nil {
		t.Fatal("Navigation in details mode should return a command")
	}
	msg := cmd()
	detailsMsg, ok := msg.(WorktreeDetailsMsg)
	if !ok {
		t.Fatalf("Expected WorktreeDetailsMsg, got %T", msg)
	}
	if detailsMsg.Worktree == nil || detailsMsg.Worktree.Name != "wt2" {
		t.Error("Details should be for wt2")
	}

	// Navigate up - should send WorktreeDetailsMsg for new selection
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.selected != 0 {
		t.Errorf("Expected selection to move to 0, got %d", m.selected)
	}
	if cmd == nil {
		t.Fatal("Navigation in details mode should return a command")
	}
	msg = cmd()
	detailsMsg, ok = msg.(WorktreeDetailsMsg)
	if !ok {
		t.Fatalf("Expected WorktreeDetailsMsg, got %T", msg)
	}
	if detailsMsg.Worktree == nil || detailsMsg.Worktree.Name != "wt1" {
		t.Error("Details should be for wt1")
	}
}

func TestNavigationDoesNotUpdateDetailsWhenNotInDetailsMode(t *testing.T) {
	m := Model{
		selected:    0,
		detailsMode: false,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
			{Name: "wt2", Path: "/test/wt2"},
		},
	}

	// Navigate down - should not send WorktreeDetailsMsg
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.selected != 1 {
		t.Errorf("Expected selection to move to 1, got %d", m.selected)
	}
	// cmd should be nil because we're not in details mode
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(WorktreeDetailsMsg); ok {
			t.Error("Should not send WorktreeDetailsMsg when not in details mode")
		}
	}
}

func TestGShortcutUpdatesDetailsInDetailsMode(t *testing.T) {
	m := Model{
		selected:    2,
		detailsMode: true,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
			{Name: "wt2", Path: "/test/wt2"},
			{Name: "wt3", Path: "/test/wt3"},
		},
	}

	// Press 'g' to go to top - should send WorktreeDetailsMsg
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.selected != 0 {
		t.Errorf("Expected selection to move to 0, got %d", m.selected)
	}
	if cmd == nil {
		t.Fatal("'g' in details mode should return a command")
	}
	msg := cmd()
	detailsMsg, ok := msg.(WorktreeDetailsMsg)
	if !ok {
		t.Fatalf("Expected WorktreeDetailsMsg, got %T", msg)
	}
	if detailsMsg.Worktree == nil || detailsMsg.Worktree.Name != "wt1" {
		t.Error("Details should be for wt1")
	}
}

func TestShiftGShortcutUpdatesDetailsInDetailsMode(t *testing.T) {
	m := Model{
		selected:    0,
		detailsMode: true,
		worktrees: []WorktreeItem{
			{Name: "wt1", Path: "/test/wt1"},
			{Name: "wt2", Path: "/test/wt2"},
			{Name: "wt3", Path: "/test/wt3"},
		},
	}

	// Press 'G' to go to bottom - should send WorktreeDetailsMsg
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.selected != 2 {
		t.Errorf("Expected selection to move to 2, got %d", m.selected)
	}
	if cmd == nil {
		t.Fatal("'G' in details mode should return a command")
	}
	msg := cmd()
	detailsMsg, ok := msg.(WorktreeDetailsMsg)
	if !ok {
		t.Fatalf("Expected WorktreeDetailsMsg, got %T", msg)
	}
	if detailsMsg.Worktree == nil || detailsMsg.Worktree.Name != "wt3" {
		t.Error("Details should be for wt3")
	}
}

func TestCloseWorktreeDetailsMsg(t *testing.T) {
	// Test that the message type exists and can be created
	msg := CloseWorktreeDetailsMsg{}
	if msg != (CloseWorktreeDetailsMsg{}) {
		t.Error("CloseWorktreeDetailsMsg should be an empty struct")
	}
}
