package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGetTheme(t *testing.T) {
	tests := []struct {
		name         string
		expectedName string
	}{
		{"default", "default"},
		{"dark", "dark"},
		{"light", "light"},
		{"nonexistent", "default"},
		{"", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTheme(tt.name)
			if got.Name != tt.expectedName {
				t.Errorf("GetTheme(%q).Name = %v, want %v", tt.name, got.Name, tt.expectedName)
			}
			// Verify theme has all required colors set
			if got.PrimaryText == nil {
				t.Error("Theme PrimaryText should not be nil")
			}
			if got.Background == nil {
				t.Error("Theme Background should not be nil")
			}
		})
	}
}

func TestGetAgentColor(t *testing.T) {
	tests := []struct {
		agent string
		want  string
	}{
		{"droid", "#ff8c00"},
		{"claude", "#d2b48c"},
		{"codex", "#000000"},
		{"unknown", "#7aa2f7"},
		{"", "#7aa2f7"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			got := GetAgentColor(tt.agent)
			if got == nil {
				t.Error("GetAgentColor should not return nil")
			}
			// Check if it's a Color type and matches expected value
			if color, ok := got.(lipgloss.Color); ok {
				if string(color) != tt.want {
					t.Errorf("GetAgentColor(%q) = %v, want %v", tt.agent, color, tt.want)
				}
			}
		})
	}
}

func TestThemesMapContainsExpectedThemes(t *testing.T) {
	expectedThemes := []string{"default", "dark", "light"}

	for _, name := range expectedThemes {
		if _, ok := Themes[name]; !ok {
			t.Errorf("Themes map missing expected theme: %q", name)
		}
	}

	// Verify each theme has a proper Name field
	for name, theme := range Themes {
		if theme.Name != name {
			t.Errorf("Theme %q has Name field %q, should match key", name, theme.Name)
		}
	}
}

func TestLogoColor(t *testing.T) {
	if LogoColor == "" {
		t.Error("LogoColor should not be empty")
	}
	expectedOrange := lipgloss.Color("#FF8C00")
	if LogoColor != expectedOrange {
		t.Errorf("LogoColor = %v, want %v", LogoColor, expectedOrange)
	}
}

func TestGetStatusColors(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
	}{
		{"default theme", "default"},
		{"light theme", "light"},
		{"dark theme", "dark"},
		{"unknown theme", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colors := GetStatusColors(tt.themeName)
			if colors.Active == nil {
				t.Error("StatusColors.Active should not be nil")
			}
			if colors.Running == nil {
				t.Error("StatusColors.Running should not be nil")
			}
			if colors.Stale == nil {
				t.Error("StatusColors.Stale should not be nil")
			}
		})
	}
}

func TestThemeColorTypes(t *testing.T) {
	t.Run("DefaultTheme uses AdaptiveColor", func(t *testing.T) {
		theme := DefaultTheme
		if _, ok := theme.PrimaryText.(lipgloss.AdaptiveColor); !ok {
			t.Error("DefaultTheme.PrimaryText should be AdaptiveColor")
		}
	})

	t.Run("LightTheme uses fixed Color", func(t *testing.T) {
		theme := LightTheme
		if _, ok := theme.PrimaryText.(lipgloss.Color); !ok {
			t.Error("LightTheme.PrimaryText should be fixed Color")
		}
		if _, ok := theme.Background.(lipgloss.Color); !ok {
			t.Error("LightTheme.Background should be fixed Color")
		}
	})

	t.Run("DarkTheme uses fixed Color", func(t *testing.T) {
		theme := DarkTheme
		if _, ok := theme.PrimaryText.(lipgloss.Color); !ok {
			t.Error("DarkTheme.PrimaryText should be fixed Color")
		}
		if _, ok := theme.Background.(lipgloss.Color); !ok {
			t.Error("DarkTheme.Background should be fixed Color")
		}
	})
}

func TestAllThemesHaveRequiredColors(t *testing.T) {
	for name, theme := range Themes {
		t.Run(name, func(t *testing.T) {
			requiredColors := map[string]ColorValue{
				"Background":         theme.Background,
				"SelectedBackground": theme.SelectedBackground,
				"PrimaryBorder":      theme.PrimaryBorder,
				"SecondaryBorder":    theme.SecondaryBorder,
				"FaintBorder":        theme.FaintBorder,
				"PrimaryText":        theme.PrimaryText,
				"SecondaryText":      theme.SecondaryText,
				"FaintText":          theme.FaintText,
				"InvertedText":       theme.InvertedText,
				"SuccessText":        theme.SuccessText,
				"WarningText":        theme.WarningText,
				"ErrorText":          theme.ErrorText,
			}

			for colorName, color := range requiredColors {
				if color == nil {
					t.Errorf("Theme %q missing %s", name, colorName)
				}
			}
		})
	}
}

func TestStatusColorConsistency(t *testing.T) {
	// Test that each theme returns consistent status colors
	themes := []string{"default", "light", "dark"}
	for _, themeName := range themes {
		t.Run(themeName, func(t *testing.T) {
			colors1 := GetStatusColors(themeName)
			colors2 := GetStatusColors(themeName)

			// Should return same values each time
			if colors1.Active != colors2.Active {
				t.Error("GetStatusColors should return consistent Active color")
			}
			if colors1.Running != colors2.Running {
				t.Error("GetStatusColors should return consistent Running color")
			}
		})
	}
}
