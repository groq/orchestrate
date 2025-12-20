// Package context provides shared context for TUI components.
package context

import (
	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// ProgramContext holds shared state accessible by all TUI components.
type ProgramContext struct {
	// Screen dimensions
	ScreenWidth  int
	ScreenHeight int

	// Content area dimensions (excluding headers/footers)
	MainContentWidth  int
	MainContentHeight int

	// Configuration
	AppSettings  *config.AppSettings
	PresetConfig *config.Config
	DataDir      string

	// UI State
	View          constants.ViewType
	SidebarOpen   bool
	HelpExpanded  bool
	Error         error
	StatusMessage string
	StatusIsError bool

	// Theme
	Theme  theme.Theme
	Styles Styles

	// Version info
	Version string
}

// NewProgramContext creates a new program context with default values.
func NewProgramContext(dataDir string, appSettings *config.AppSettings, presetConfig *config.Config) *ProgramContext {
	t := theme.GetTheme(appSettings.UI.Theme)
	ctx := &ProgramContext{
		DataDir:      dataDir,
		AppSettings:  appSettings,
		PresetConfig: presetConfig,
		View:         constants.WorktreesView,
		SidebarOpen:  false,
		HelpExpanded: false,
		Theme:        t,
		Version:      "dev",
	}
	ctx.Styles = InitStyles(t)
	return ctx
}

// UpdateWindowSize updates the context when window size changes.
func (ctx *ProgramContext) UpdateWindowSize(msg tea.WindowSizeMsg) {
	ctx.ScreenWidth = msg.Width
	ctx.ScreenHeight = msg.Height
	ctx.syncContentDimensions()
}

// syncContentDimensions calculates main content area dimensions.
func (ctx *ProgramContext) syncContentDimensions() {
	// Rigid header height
	headerHeight := constants.HeaderHeight

	// Main content height is remaining screen minus header
	ctx.MainContentHeight = ctx.ScreenHeight - headerHeight
	ctx.MainContentWidth = ctx.ScreenWidth

	if ctx.SidebarOpen {
		ctx.MainContentWidth = ctx.ScreenWidth - constants.SidebarWidth
	}
}

// ToggleSidebar toggles the sidebar visibility.
func (ctx *ProgramContext) ToggleSidebar() {
	ctx.SidebarOpen = !ctx.SidebarOpen
	ctx.syncContentDimensions()
}

// ToggleHelp toggles the expanded help view.
func (ctx *ProgramContext) ToggleHelp() {
	ctx.HelpExpanded = !ctx.HelpExpanded
	ctx.syncContentDimensions()
}

// SetView changes the current view.
func (ctx *ProgramContext) SetView(view constants.ViewType) {
	ctx.View = view
	ctx.ClearStatus()
}

// SetStatus sets the status message.
func (ctx *ProgramContext) SetStatus(message string, isError bool) {
	ctx.StatusMessage = message
	ctx.StatusIsError = isError
}

// ClearStatus clears the status message.
func (ctx *ProgramContext) ClearStatus() {
	ctx.StatusMessage = ""
	ctx.StatusIsError = false
	ctx.Error = nil
}

// SetError sets an error state.
func (ctx *ProgramContext) SetError(err error) {
	ctx.Error = err
	if err != nil {
		ctx.StatusMessage = err.Error()
		ctx.StatusIsError = true
	}
}

// UpdateTheme updates the theme.
func (ctx *ProgramContext) UpdateTheme(themeName string) {
	ctx.Theme = theme.GetTheme(themeName)
	ctx.Styles = InitStyles(ctx.Theme)
}

// GetPresetNames returns a list of preset names.
func (ctx *ProgramContext) GetPresetNames() []string {
	if ctx.PresetConfig == nil {
		return nil
	}
	names := make([]string, 0, len(ctx.PresetConfig.Presets))
	for name := range ctx.PresetConfig.Presets {
		names = append(names, name)
	}
	return names
}

// GetDefaultPreset returns the default preset name.
func (ctx *ProgramContext) GetDefaultPreset() string {
	if ctx.PresetConfig != nil && ctx.PresetConfig.Default != "" {
		return ctx.PresetConfig.Default
	}
	if ctx.AppSettings != nil && ctx.AppSettings.Session.DefaultPreset != "" {
		return ctx.AppSettings.Session.DefaultPreset
	}
	return "default"
}
