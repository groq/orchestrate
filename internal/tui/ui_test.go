package tui

import (
	"testing"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	appSettings := config.DefaultAppSettings()
	presetConfig := &config.Config{
		Default: "test",
		Presets: map[string]config.Preset{
			"test": {{Agent: "droid"}},
		},
	}

	model := NewModel("/tmp/test", appSettings, presetConfig)

	if model.ctx == nil {
		t.Fatal("ctx should not be nil")
	}
	if model.ctx.DataDir != "/tmp/test" {
		t.Errorf("DataDir = %q, want %q", model.ctx.DataDir, "/tmp/test")
	}
	if model.ctx.View != constants.WorktreesView {
		t.Errorf("View = %v, want WorktreesView", model.ctx.View)
	}
	if model.ready {
		t.Error("Model should not be ready before receiving WindowSizeMsg")
	}
}

func TestModel_HeaderNavigation(t *testing.T) {
	appSettings := config.DefaultAppSettings()
	model := NewModel("/tmp", appSettings, nil)

	// Make ready
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	m, _ := model.Update(msg)
	model = m.(Model)

	if model.focus != FocusContent {
		t.Error("Initial focus should be FocusContent")
	}

	// Move up to header
	// In WorktreesView, up arrow when at top (selected=0) moves to header
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = m.(Model)

	if model.focus != FocusHeader {
		t.Error("Focus should be FocusHeader after 'up' at top of content")
	}

	// Test page switching in header
	initialView := model.ctx.View
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyRight})
	model = m.(Model)

	if model.ctx.View == initialView {
		t.Error("View should change when pressing 'right' in header")
	}

	// Move back down to content
	m, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = m.(Model)

	if model.focus != FocusContent {
		t.Error("Focus should be FocusContent after 'down' in header")
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	model := NewModel("/tmp", config.DefaultAppSettings(), nil)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, _ := model.Update(msg)

	m := newModel.(Model)
	if !m.ready {
		t.Error("Model should be ready after WindowSizeMsg")
	}
	if m.ctx.ScreenWidth != 100 {
		t.Errorf("ScreenWidth = %d, want 100", m.ctx.ScreenWidth)
	}
	if m.ctx.ScreenHeight != 50 {
		t.Errorf("ScreenHeight = %d, want 50", m.ctx.ScreenHeight)
	}
}

func TestModel_Update_BackNavigation(t *testing.T) {
	model := NewModel("/tmp", config.DefaultAppSettings(), nil)
	newModel, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	model = newModel.(Model)

	// Go to launch view via helper (Tab is now handled globally)
	model.ctx.SetView(constants.LaunchView)

	if model.ctx.View != constants.LaunchView {
		t.Fatal("Should be in launch view")
	}

	// Press back (Esc)
	newModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = newModel.(Model)

	// Default behavior for Esc when not consumed is back to Worktrees
	if model.ctx.View != constants.WorktreesView {
		t.Errorf("View = %v, want WorktreesView after back", model.ctx.View)
	}
}
