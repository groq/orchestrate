package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// TestThemeIntegration verifies that themes work properly in practice.
func TestThemeIntegration(t *testing.T) {
	t.Run("Light theme has light colors", func(t *testing.T) {
		theme := GetTheme("light")

		// Background should be white or near-white
		if bg, ok := theme.Background.(lipgloss.Color); ok {
			if string(bg) != "#ffffff" {
				t.Errorf("Light theme background = %v, want #ffffff", bg)
			}
		} else {
			t.Error("Light theme Background should be fixed Color, not AdaptiveColor")
		}

		// Text should be dark
		if text, ok := theme.PrimaryText.(lipgloss.Color); ok {
			if string(text) != "#1a1a1a" {
				t.Errorf("Light theme text = %v, want #1a1a1a", text)
			}
		}
	})

	t.Run("Dark theme has dark colors", func(t *testing.T) {
		theme := GetTheme("dark")

		// Background should be dark
		if bg, ok := theme.Background.(lipgloss.Color); ok {
			if string(bg) != "#1a1b26" {
				t.Errorf("Dark theme background = %v, want #1a1b26", bg)
			}
		} else {
			t.Error("Dark theme Background should be fixed Color, not AdaptiveColor")
		}

		// Text should be light
		if text, ok := theme.PrimaryText.(lipgloss.Color); ok {
			if string(text) != "#c0caf5" {
				t.Errorf("Dark theme text = %v, want #c0caf5", text)
			}
		}
	})

	t.Run("Default theme is adaptive", func(t *testing.T) {
		theme := GetTheme("default")

		// Should use AdaptiveColor
		if _, ok := theme.Background.(lipgloss.AdaptiveColor); !ok {
			t.Error("Default theme should use AdaptiveColor")
		}

		if _, ok := theme.PrimaryText.(lipgloss.AdaptiveColor); !ok {
			t.Error("Default theme should use AdaptiveColor")
		}
	})
}

func TestThemeSwitching(t *testing.T) {
	themes := []string{"default", "light", "dark"}

	for _, themeName := range themes {
		t.Run(themeName, func(t *testing.T) {
			theme := GetTheme(themeName)

			// Verify theme can be retrieved
			if theme.Name != themeName {
				t.Errorf("Got theme %q, want %q", theme.Name, themeName)
			}

			// Verify all colors are set
			if theme.PrimaryText == nil {
				t.Error("PrimaryText should be set")
			}
			if theme.SuccessText == nil {
				t.Error("SuccessText should be set")
			}
			if theme.ErrorText == nil {
				t.Error("ErrorText should be set")
			}

			// Verify status colors work for this theme
			statusColors := GetStatusColors(themeName)
			if statusColors.Active == nil {
				t.Error("Status Active color should be set")
			}
		})
	}
}

func TestColorValueInterface(t *testing.T) {
	t.Run("lipgloss.Color implements ColorValue", func(t *testing.T) {
		var _ ColorValue = lipgloss.Color("#FF0000")
	})

	t.Run("lipgloss.AdaptiveColor implements ColorValue", func(t *testing.T) {
		var _ ColorValue = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}
	})
}

func TestThemeConsistency(t *testing.T) {
	// Test that retrieving a theme multiple times returns consistent results
	t.Run("Multiple GetTheme calls return same theme", func(t *testing.T) {
		theme1 := GetTheme("dark")
		theme2 := GetTheme("dark")

		if theme1.Name != theme2.Name {
			t.Error("Theme name should be consistent")
		}

		// Compare a specific color to ensure it's the same theme
		if theme1.PrimaryText != theme2.PrimaryText {
			t.Error("Theme colors should be consistent across calls")
		}
	})
}

func TestStatusColorsForAllThemes(t *testing.T) {
	themes := []string{"default", "light", "dark", "nonexistent"}

	for _, themeName := range themes {
		t.Run(themeName, func(t *testing.T) {
			colors := GetStatusColors(themeName)

			// All status colors must be non-nil
			requiredColors := map[string]ColorValue{
				"Open":    colors.Open,
				"Closed":  colors.Closed,
				"Active":  colors.Active,
				"Stale":   colors.Stale,
				"Running": colors.Running,
			}

			for name, color := range requiredColors {
				if color == nil {
					t.Errorf("Status color %q should not be nil for theme %q", name, themeName)
				}
			}
		})
	}
}

func TestAgentColorsConsistency(t *testing.T) {
	agents := []string{"claude", "codex", "droid", "unknown"}

	for _, agent := range agents {
		t.Run(agent, func(t *testing.T) {
			color1 := GetAgentColor(agent)
			color2 := GetAgentColor(agent)

			if color1 != color2 {
				t.Errorf("GetAgentColor(%q) should return consistent colors", agent)
			}

			// Verify it's a fixed color (not adaptive)
			if _, ok := color1.(lipgloss.Color); !ok {
				t.Errorf("Agent color for %q should be fixed Color", agent)
			}
		})
	}
}

func BenchmarkGetTheme(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetTheme("dark")
	}
}

func BenchmarkGetAgentColor(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetAgentColor("claude")
	}
}

func BenchmarkGetStatusColors(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetStatusColors("dark")
	}
}
