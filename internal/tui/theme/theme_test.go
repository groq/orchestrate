package theme

import (
	"testing"
)

func TestGetTheme(t *testing.T) {
	tests := []struct {
		name     string
		expected Theme
	}{
		{"default", DefaultTheme},
		{"dark", DarkTheme},
		{"light", LightTheme},
		{"nonexistent", DefaultTheme},
		{"", DefaultTheme},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTheme(tt.name)
			// Compare primary text as a proxy for theme identity
			if got.PrimaryText != tt.expected.PrimaryText {
				t.Errorf("GetTheme(%q) returned unexpected theme", tt.name)
			}
		})
	}
}

func TestGetAgentColor(t *testing.T) {
	tests := []struct {
		agent    string
		expected string
	}{
		{"droid", DefaultAgentColors.Droid.Dark},
		{"claude", DefaultAgentColors.Claude.Dark},
		{"codex", DefaultAgentColors.Codex.Dark},
		{"unknown", DefaultAgentColors.Default.Dark},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			got := GetAgentColor(tt.agent)
			if got.Dark != tt.expected {
				t.Errorf("GetAgentColor(%q) = %v, want %v", tt.agent, got.Dark, tt.expected)
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
}

func TestLogoColor(t *testing.T) {
	if LogoColor == "" {
		t.Error("LogoColor should not be empty")
	}
}

func TestDefaultStatusColors(t *testing.T) {
	if DefaultStatusColors.Active.Dark == "" {
		t.Error("DefaultStatusColors.Active should not be empty")
	}
	if DefaultStatusColors.Running.Dark == "" {
		t.Error("DefaultStatusColors.Running should not be empty")
	}
}
