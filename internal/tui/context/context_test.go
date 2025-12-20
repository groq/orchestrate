package context

import (
	"testing"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewProgramContext(t *testing.T) {
	appSettings := config.DefaultAppSettings()
	presetConfig := &config.Config{
		Default: "test",
		Presets: map[string]config.Preset{
			"test": {{Agent: "droid"}},
		},
	}

	ctx := NewProgramContext("/tmp/test", appSettings, presetConfig)

	if ctx.DataDir != "/tmp/test" {
		t.Errorf("DataDir = %q, want %q", ctx.DataDir, "/tmp/test")
	}
	if ctx.View != constants.WorktreesView {
		t.Errorf("View = %v, want %v", ctx.View, constants.WorktreesView)
	}
	if ctx.SidebarOpen {
		t.Error("SidebarOpen should be false by default")
	}
}

func TestProgramContext_UpdateWindowSize(t *testing.T) {
	ctx := NewProgramContext("/tmp", config.DefaultAppSettings(), nil)

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	ctx.UpdateWindowSize(msg)

	if ctx.ScreenWidth != 120 {
		t.Errorf("ScreenWidth = %d, want 120", ctx.ScreenWidth)
	}
	if ctx.ScreenHeight != 40 {
		t.Errorf("ScreenHeight = %d, want 40", ctx.ScreenHeight)
	}
	if ctx.MainContentHeight <= 0 {
		t.Error("MainContentHeight should be positive")
	}
}

func TestProgramContext_ToggleSidebar(t *testing.T) {
	ctx := NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 120, Height: 40})

	if ctx.SidebarOpen {
		t.Error("SidebarOpen should start false")
	}

	ctx.ToggleSidebar()
	if !ctx.SidebarOpen {
		t.Error("SidebarOpen should be true after toggle")
	}

	ctx.ToggleSidebar()
	if ctx.SidebarOpen {
		t.Error("SidebarOpen should be false after second toggle")
	}
}

func TestProgramContext_ToggleHelp(t *testing.T) {
	ctx := NewProgramContext("/tmp", config.DefaultAppSettings(), nil)

	if ctx.HelpExpanded {
		t.Error("HelpExpanded should start false")
	}

	ctx.ToggleHelp()
	if !ctx.HelpExpanded {
		t.Error("HelpExpanded should be true after toggle")
	}
}

func TestProgramContext_SetView(t *testing.T) {
	ctx := NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.SetStatus("test", false)

	ctx.SetView(constants.SettingsView)

	if ctx.View != constants.SettingsView {
		t.Errorf("View = %v, want %v", ctx.View, constants.SettingsView)
	}
	if ctx.StatusMessage != "" {
		t.Error("Status should be cleared when view changes")
	}
}

func TestProgramContext_SetStatus(t *testing.T) {
	ctx := NewProgramContext("/tmp", config.DefaultAppSettings(), nil)

	ctx.SetStatus("Test message", false)
	if ctx.StatusMessage != "Test message" {
		t.Errorf("StatusMessage = %q, want %q", ctx.StatusMessage, "Test message")
	}
	if ctx.StatusIsError {
		t.Error("StatusIsError should be false")
	}

	ctx.SetStatus("Error message", true)
	if !ctx.StatusIsError {
		t.Error("StatusIsError should be true")
	}
}

func TestProgramContext_GetPresetNames(t *testing.T) {
	presetConfig := &config.Config{
		Presets: map[string]config.Preset{
			"alpha": {{Agent: "droid"}},
			"beta":  {{Agent: "claude"}},
		},
	}
	ctx := NewProgramContext("/tmp", config.DefaultAppSettings(), presetConfig)

	names := ctx.GetPresetNames()
	if len(names) != 2 {
		t.Errorf("len(names) = %d, want 2", len(names))
	}
}

func TestProgramContext_GetPresetNames_Nil(t *testing.T) {
	ctx := NewProgramContext("/tmp", config.DefaultAppSettings(), nil)

	names := ctx.GetPresetNames()
	if names != nil {
		t.Errorf("names should be nil when PresetConfig is nil")
	}
}

func TestProgramContext_GetDefaultPreset(t *testing.T) {
	tests := []struct {
		name         string
		presetConfig *config.Config
		appSettings  *config.AppSettings
		expected     string
	}{
		{
			name:         "from preset config",
			presetConfig: &config.Config{Default: "preset-default"},
			appSettings:  config.DefaultAppSettings(),
			expected:     "preset-default",
		},
		{
			name:         "from app settings",
			presetConfig: nil,
			appSettings:  &config.AppSettings{Session: config.SessionSettings{DefaultPreset: "app-default"}},
			expected:     "app-default",
		},
		{
			name:         "fallback to default",
			presetConfig: nil,
			appSettings:  &config.AppSettings{},
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewProgramContext("/tmp", tt.appSettings, tt.presetConfig)
			got := ctx.GetDefaultPreset()
			if got != tt.expected {
				t.Errorf("GetDefaultPreset() = %q, want %q", got, tt.expected)
			}
		})
	}
}

