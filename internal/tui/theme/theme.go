// Package theme provides theming support for the TUI.
package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines color and style configuration.
// Modeled after gh-dash's theming system for consistency.
type Theme struct {
	// Background colors
	Background         lipgloss.AdaptiveColor
	SelectedBackground lipgloss.AdaptiveColor

	// Border colors
	PrimaryBorder   lipgloss.AdaptiveColor
	SecondaryBorder lipgloss.AdaptiveColor
	FaintBorder     lipgloss.AdaptiveColor

	// Text colors
	PrimaryText   lipgloss.AdaptiveColor
	SecondaryText lipgloss.AdaptiveColor
	FaintText     lipgloss.AdaptiveColor
	InvertedText  lipgloss.AdaptiveColor

	// Status colors
	SuccessText lipgloss.AdaptiveColor
	WarningText lipgloss.AdaptiveColor
	ErrorText   lipgloss.AdaptiveColor
}

// LogoColor is the primary branding color (orange for Orchestrate).
var LogoColor = lipgloss.Color("#FF8C00")

// DefaultTheme provides the default orchestrate theme (dark).
var DefaultTheme = Theme{
	Background:         lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#1a1b26"},
	SelectedBackground: lipgloss.AdaptiveColor{Light: "#e1e2e7", Dark: "#282a36"},
	PrimaryBorder:      lipgloss.AdaptiveColor{Light: "#013", Dark: "#44475a"},
	SecondaryBorder:    lipgloss.AdaptiveColor{Light: "#008", Dark: "#3d59a1"}, // blue
	FaintBorder:        lipgloss.AdaptiveColor{Light: "#ddd", Dark: "#21222c"},

	PrimaryText:   lipgloss.AdaptiveColor{Light: "#000", Dark: "#f8f8f2"},
	SecondaryText: lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#7dcfff"}, // blue
	FaintText:     lipgloss.AdaptiveColor{Light: "#999", Dark: "#565f89"},    // grey
	InvertedText:  lipgloss.AdaptiveColor{Light: "#fff", Dark: "#282a36"},

	SuccessText: lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#7dcfff"}, // blue
	WarningText: lipgloss.AdaptiveColor{Light: "#f57c00", Dark: "#ffb86c"}, // orange
	ErrorText:   lipgloss.AdaptiveColor{Light: "#c62828", Dark: "#f7768e"}, // red
}

// LightTheme provides a light version.
var LightTheme = Theme{
	Background:         lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},
	SelectedBackground: lipgloss.AdaptiveColor{Light: "#eeeeee", Dark: "#eeeeee"},
	PrimaryBorder:      lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#cccccc"},
	SecondaryBorder:    lipgloss.AdaptiveColor{Light: "#aaaaaa", Dark: "#aaaaaa"},
	FaintBorder:        lipgloss.AdaptiveColor{Light: "#f0f0f0", Dark: "#f0f0f0"},

	PrimaryText:   lipgloss.AdaptiveColor{Light: "#111111", Dark: "#111111"},
	SecondaryText: lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#0066cc"},
	FaintText:     lipgloss.AdaptiveColor{Light: "#888888", Dark: "#888888"},
	InvertedText:  lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#ffffff"},

	SuccessText: lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#0066cc"},
	WarningText: lipgloss.AdaptiveColor{Light: "#f57c00", Dark: "#f57c00"},
	ErrorText:   lipgloss.AdaptiveColor{Light: "#c62828", Dark: "#c62828"},
}

// DarkTheme provides a dark version.
var DarkTheme = Theme{
	Background:         lipgloss.AdaptiveColor{Light: "#1a1b26", Dark: "#1a1b26"},
	SelectedBackground: lipgloss.AdaptiveColor{Light: "#282a36", Dark: "#282a36"},
	PrimaryBorder:      lipgloss.AdaptiveColor{Light: "#44475a", Dark: "#44475a"},
	SecondaryBorder:    lipgloss.AdaptiveColor{Light: "#3d59a1", Dark: "#3d59a1"},
	FaintBorder:        lipgloss.AdaptiveColor{Light: "#21222c", Dark: "#21222c"},

	PrimaryText:   lipgloss.AdaptiveColor{Light: "#f8f8f2", Dark: "#f8f8f2"},
	SecondaryText: lipgloss.AdaptiveColor{Light: "#7aa2f7", Dark: "#7aa2f7"},
	FaintText:     lipgloss.AdaptiveColor{Light: "#565f89", Dark: "#565f89"},
	InvertedText:  lipgloss.AdaptiveColor{Light: "#282a36", Dark: "#282a36"},

	SuccessText: lipgloss.AdaptiveColor{Light: "#7dcfff", Dark: "#7dcfff"},
	WarningText: lipgloss.AdaptiveColor{Light: "#ffb86c", Dark: "#ffb86c"},
	ErrorText:   lipgloss.AdaptiveColor{Light: "#f7768e", Dark: "#f7768e"},
}

// StatusColors provides semantic status colors.
type StatusColors struct {
	Open    lipgloss.AdaptiveColor
	Closed  lipgloss.AdaptiveColor
	Active  lipgloss.AdaptiveColor
	Stale   lipgloss.AdaptiveColor
	Running lipgloss.AdaptiveColor
}

// DefaultStatusColors for worktree and session status.
var DefaultStatusColors = StatusColors{
	Open:    lipgloss.AdaptiveColor{Light: "#1565c0", Dark: "#7aa2f7"},
	Closed:  lipgloss.AdaptiveColor{Light: "#c62828", Dark: "#f7768e"},
	Active:  lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#7dcfff"},
	Stale:   lipgloss.AdaptiveColor{Light: "#999", Dark: "#565f89"},
	Running: lipgloss.AdaptiveColor{Light: "#f57c00", Dark: "#ffb86c"},
}

// AgentColors for different AI agents.
type AgentColors struct {
	Claude  lipgloss.AdaptiveColor
	Codex   lipgloss.AdaptiveColor
	Droid   lipgloss.AdaptiveColor
	Default lipgloss.AdaptiveColor
}

// DefaultAgentColors for visual identification.
var DefaultAgentColors = AgentColors{
	Claude:  lipgloss.AdaptiveColor{Light: "#a08060", Dark: "#d2b48c"},
	Codex:   lipgloss.AdaptiveColor{Light: "#00897b", Dark: "#4db6ac"},
	Droid:   lipgloss.AdaptiveColor{Light: "#ff5722", Dark: "#ff8c00"},
	Default: lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#7dcfff"},
}

// GetAgentColor returns the color for an agent name.
func GetAgentColor(agent string) lipgloss.AdaptiveColor {
	switch agent {
	case "claude":
		return DefaultAgentColors.Claude
	case "codex":
		return DefaultAgentColors.Codex
	case "droid":
		return DefaultAgentColors.Droid
	default:
		return DefaultAgentColors.Default
	}
}

// Themes maps theme names to their definitions.
var Themes = map[string]Theme{
	"default": DefaultTheme,
	"dark":    DarkTheme,
	"light":   LightTheme,
}

// GetTheme returns the theme by name, defaulting to DefaultTheme.
func GetTheme(name string) Theme {
	if t, ok := Themes[name]; ok {
		return t
	}
	return DefaultTheme
}
