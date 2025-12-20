// Package theme provides theming support for the TUI.
package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// ColorValue can be either a fixed color or an adaptive color.
type ColorValue interface {
	lipgloss.TerminalColor
}

// Theme defines color and style configuration.
// Modeled after gh-dash's theming system for consistency.
type Theme struct {
	Name string // Theme name for identification

	// Background colors
	Background         ColorValue
	SelectedBackground ColorValue

	// Border colors
	PrimaryBorder   ColorValue
	SecondaryBorder ColorValue
	FaintBorder     ColorValue

	// Text colors
	PrimaryText   ColorValue
	SecondaryText ColorValue
	FaintText     ColorValue
	InvertedText  ColorValue

	// Status colors
	SuccessText ColorValue
	WarningText ColorValue
	ErrorText   ColorValue
}

// LogoColor is the primary branding color (orange for Orchestrate).
var LogoColor = lipgloss.Color("#FF8C00")

// DefaultTheme provides the default orchestrate theme (adaptive - respects terminal appearance).
var DefaultTheme = Theme{
	Name:               "default",
	Background:         lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#1a1b26"},
	SelectedBackground: lipgloss.AdaptiveColor{Light: "#e1e2e7", Dark: "#282a36"},
	PrimaryBorder:      lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#44475a"},
	SecondaryBorder:    lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#3d59a1"},
	FaintBorder:        lipgloss.AdaptiveColor{Light: "#e0e0e0", Dark: "#21222c"},

	PrimaryText:   lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f8f8f2"},
	SecondaryText: lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#7aa2f7"},
	FaintText:     lipgloss.AdaptiveColor{Light: "#888888", Dark: "#565f89"},
	InvertedText:  lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#1a1b26"},

	SuccessText: lipgloss.AdaptiveColor{Light: "#00aa00", Dark: "#9ece6a"},
	WarningText: lipgloss.AdaptiveColor{Light: "#ff8800", Dark: "#ffb86c"},
	ErrorText:   lipgloss.AdaptiveColor{Light: "#dd0000", Dark: "#f7768e"},
}

// LightTheme provides a fixed light theme (ignores terminal appearance).
var LightTheme = Theme{
	Name:               "light",
	Background:         lipgloss.Color("#ffffff"),
	SelectedBackground: lipgloss.Color("#f0f0f0"),
	PrimaryBorder:      lipgloss.Color("#cccccc"),
	SecondaryBorder:    lipgloss.Color("#aaaaaa"),
	FaintBorder:        lipgloss.Color("#e8e8e8"),

	PrimaryText:   lipgloss.Color("#1a1a1a"),
	SecondaryText: lipgloss.Color("#0066cc"),
	FaintText:     lipgloss.Color("#888888"),
	InvertedText:  lipgloss.Color("#ffffff"),

	SuccessText: lipgloss.Color("#008800"),
	WarningText: lipgloss.Color("#ff8800"),
	ErrorText:   lipgloss.Color("#cc0000"),
}

// DarkTheme provides a fixed dark theme (ignores terminal appearance).
var DarkTheme = Theme{
	Name:               "dark",
	Background:         lipgloss.Color("#1a1b26"),
	SelectedBackground: lipgloss.Color("#24283b"),
	PrimaryBorder:      lipgloss.Color("#414868"),
	SecondaryBorder:    lipgloss.Color("#3d59a1"),
	FaintBorder:        lipgloss.Color("#292e42"),

	PrimaryText:   lipgloss.Color("#c0caf5"),
	SecondaryText: lipgloss.Color("#7aa2f7"),
	FaintText:     lipgloss.Color("#565f89"),
	InvertedText:  lipgloss.Color("#1a1b26"),

	SuccessText: lipgloss.Color("#9ece6a"),
	WarningText: lipgloss.Color("#e0af68"),
	ErrorText:   lipgloss.Color("#f7768e"),
}

// StatusColors provides semantic status colors for a specific theme.
type StatusColors struct {
	Open    ColorValue
	Closed  ColorValue
	Active  ColorValue
	Stale   ColorValue
	Running ColorValue
}

// GetStatusColors returns status colors for a theme.
func GetStatusColors(themeName string) StatusColors {
	switch themeName {
	case "light":
		return StatusColors{
			Open:    lipgloss.Color("#0066cc"),
			Closed:  lipgloss.Color("#cc0000"),
			Active:  lipgloss.Color("#0088ff"),
			Stale:   lipgloss.Color("#888888"),
			Running: lipgloss.Color("#ff8800"),
		}
	case "dark":
		return StatusColors{
			Open:    lipgloss.Color("#7aa2f7"),
			Closed:  lipgloss.Color("#f7768e"),
			Active:  lipgloss.Color("#7dcfff"),
			Stale:   lipgloss.Color("#565f89"),
			Running: lipgloss.Color("#e0af68"),
		}
	default: // "default" - adaptive
		return StatusColors{
			Open:    lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#7aa2f7"},
			Closed:  lipgloss.AdaptiveColor{Light: "#cc0000", Dark: "#f7768e"},
			Active:  lipgloss.AdaptiveColor{Light: "#0088ff", Dark: "#7dcfff"},
			Stale:   lipgloss.AdaptiveColor{Light: "#888888", Dark: "#565f89"},
			Running: lipgloss.AdaptiveColor{Light: "#ff8800", Dark: "#e0af68"},
		}
	}
}

// GetAgentColor returns the color for an agent name based on theme.
func GetAgentColor(agent string) ColorValue {
	// Agent colors are always the same regardless of theme for consistency
	switch agent {
	case "claude":
		return lipgloss.Color("#d2b48c") // tan/sand
	case "codex":
		return lipgloss.Color("#000000") // black
	case "droid":
		return lipgloss.Color("#ff8c00") // orange
	default:
		return lipgloss.Color("#7aa2f7") // blue
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
