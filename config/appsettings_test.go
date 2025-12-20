package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAppSettings(t *testing.T) {
	settings := DefaultAppSettings()

	if settings == nil {
		t.Fatal("DefaultAppSettings should not return nil")
	}

	if settings.Terminal.Type != TerminalITerm2 {
		t.Errorf("Default terminal type = %v, want %v", settings.Terminal.Type, TerminalITerm2)
	}

	if settings.UI.Theme != "default" {
		t.Errorf("Default theme = %v, want 'default'", settings.UI.Theme)
	}

	if !settings.Terminal.MaximizeOnLaunch {
		t.Error("Default MaximizeOnLaunch should be true")
	}

	if !settings.UI.ShowStatusBar {
		t.Error("Default ShowStatusBar should be true")
	}
}

func TestLoadAppSettings_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()

	settings, path, err := LoadAppSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadAppSettings should not error on non-existent file: %v", err)
	}

	if settings == nil {
		t.Fatal("Should return default settings")
	}

	if settings.UI.Theme != "default" {
		t.Errorf("Default theme = %v, want 'default'", settings.UI.Theme)
	}

	if path == "" {
		t.Error("Path should not be empty")
	}
}

func TestSaveAndLoadAppSettings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create custom settings
	settings := &AppSettings{
		Terminal: TerminalSettings{
			Type:             TerminalITerm2,
			MaximizeOnLaunch: false,
		},
		UI: UISettings{
			ShowStatusBar: false,
			Theme:         "dark",
		},
		Session: SessionSettings{
			DefaultPreset:         "custom",
			AutoCleanWorktrees:    true,
			WorktreeRetentionDays: 14,
		},
	}

	// Save settings
	err := SaveAppSettings(tmpDir, settings)
	if err != nil {
		t.Fatalf("SaveAppSettings failed: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, AppSettingsFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load settings back
	loaded, _, err := LoadAppSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadAppSettings failed: %v", err)
	}

	// Verify loaded settings match
	if loaded.UI.Theme != "dark" {
		t.Errorf("Loaded theme = %v, want 'dark'", loaded.UI.Theme)
	}

	if loaded.Terminal.MaximizeOnLaunch {
		t.Error("Loaded MaximizeOnLaunch should be false")
	}

	if loaded.UI.ShowStatusBar {
		t.Error("Loaded ShowStatusBar should be false")
	}

	if loaded.Session.DefaultPreset != "custom" {
		t.Errorf("Loaded DefaultPreset = %v, want 'custom'", loaded.Session.DefaultPreset)
	}

	if !loaded.Session.AutoCleanWorktrees {
		t.Error("Loaded AutoCleanWorktrees should be true")
	}

	if loaded.Session.WorktreeRetentionDays != 14 {
		t.Errorf("Loaded WorktreeRetentionDays = %v, want 14", loaded.Session.WorktreeRetentionDays)
	}
}

func TestGetThemeOptions(t *testing.T) {
	options := GetThemeOptions()

	if len(options) == 0 {
		t.Fatal("GetThemeOptions should return at least one option")
	}

	expectedThemes := []string{"default", "dark", "light"}
	if len(options) != len(expectedThemes) {
		t.Errorf("GetThemeOptions returned %d options, want %d", len(options), len(expectedThemes))
	}

	// Verify all expected themes are present
	themeMap := make(map[string]bool)
	for _, theme := range options {
		themeMap[theme] = true
	}

	for _, expected := range expectedThemes {
		if !themeMap[expected] {
			t.Errorf("Missing expected theme: %q", expected)
		}
	}
}

func TestGetTerminalTypeOptions(t *testing.T) {
	options := GetTerminalTypeOptions()

	if len(options) == 0 {
		t.Fatal("GetTerminalTypeOptions should return at least one option")
	}

	// Verify expected options
	foundITerm := false
	foundTerminal := false

	for _, opt := range options {
		if opt == TerminalITerm2 {
			foundITerm = true
		}
		if opt == TerminalRegular {
			foundTerminal = true
		}
	}

	if !foundITerm {
		t.Error("Should include TerminalITerm2 option")
	}

	if !foundTerminal {
		t.Error("Should include TerminalRegular option")
	}
}

func TestThemePersistence(t *testing.T) {
	tmpDir := t.TempDir()

	themes := []string{"default", "light", "dark"}

	for _, theme := range themes {
		t.Run(theme, func(t *testing.T) {
			settings := DefaultAppSettings()
			settings.UI.Theme = theme

			// Save
			err := SaveAppSettings(tmpDir, settings)
			if err != nil {
				t.Fatalf("Failed to save theme %q: %v", theme, err)
			}

			// Load
			loaded, _, err := LoadAppSettings(tmpDir)
			if err != nil {
				t.Fatalf("Failed to load theme %q: %v", theme, err)
			}

			if loaded.UI.Theme != theme {
				t.Errorf("Theme persistence failed: saved %q, loaded %q", theme, loaded.UI.Theme)
			}
		})
	}
}

func TestLoadAppSettings_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, AppSettingsFileName)

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content: [[["), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = LoadAppSettings(tmpDir)
	if err == nil {
		t.Error("LoadAppSettings should error on invalid YAML")
	}
}

func TestAppSettingsFileName(t *testing.T) {
	if AppSettingsFileName != "orchestrate.yaml" {
		t.Errorf("AppSettingsFileName = %q, want 'orchestrate.yaml'", AppSettingsFileName)
	}
}

func TestTerminalTypeConstants(t *testing.T) {
	if TerminalITerm2 != "iterm2" {
		t.Errorf("TerminalITerm2 = %q, want 'iterm2'", TerminalITerm2)
	}

	if TerminalRegular != "terminal" {
		t.Errorf("TerminalRegular = %q, want 'terminal'", TerminalRegular)
	}
}
