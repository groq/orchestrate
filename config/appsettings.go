// Package config handles configuration loading for orchestrate.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// TerminalType represents the type of terminal to use
type TerminalType string

const (
	// TerminalITerm2 uses iTerm2 (macOS)
	TerminalITerm2 TerminalType = "iterm2"
	// TerminalRegular uses the default system terminal
	TerminalRegular TerminalType = "terminal"
)

// AppSettings represents app-level settings stored in orchestrate.yaml
type AppSettings struct {
	// Terminal settings
	Terminal TerminalSettings `yaml:"terminal"`

	// UI settings
	UI UISettings `yaml:"ui"`

	// Session settings
	Session SessionSettings `yaml:"session"`
}

// TerminalSettings contains terminal-related settings
type TerminalSettings struct {
	// Type of terminal to use: "iterm2" or "terminal"
	Type TerminalType `yaml:"type"`

	// MaximizeOnLaunch maximizes windows when launching sessions
	MaximizeOnLaunch bool `yaml:"maximize_on_launch"`
}

// UISettings contains UI-related settings
type UISettings struct {
	// ShowStatusBar shows the status bar in the TUI
	ShowStatusBar bool `yaml:"show_status_bar"`

	// Theme is the color theme for the TUI
	Theme string `yaml:"theme"`
}

// SessionSettings contains session-related settings
type SessionSettings struct {
	// DefaultPreset is the default preset name to use
	DefaultPreset string `yaml:"default_preset"`

	// AutoCleanWorktrees automatically cleans old worktrees
	AutoCleanWorktrees bool `yaml:"auto_clean_worktrees"`

	// WorktreeRetentionDays is how many days to keep old worktrees
	WorktreeRetentionDays int `yaml:"worktree_retention_days"`
}

// AppSettingsFileName is the name of the app settings file
const AppSettingsFileName = "orchestrate.yaml"

// DefaultAppSettings returns the default app settings
func DefaultAppSettings() *AppSettings {
	return &AppSettings{
		Terminal: TerminalSettings{
			Type:             TerminalITerm2,
			MaximizeOnLaunch: true,
		},
		UI: UISettings{
			ShowStatusBar: true,
			Theme:         "default",
		},
		Session: SessionSettings{
			DefaultPreset:         "default",
			AutoCleanWorktrees:    false,
			WorktreeRetentionDays: 7,
		},
	}
}

// LoadAppSettings loads app settings from orchestrate.yaml in the specified directory.
// Returns default settings if the file doesn't exist.
func LoadAppSettings(dir string) (*AppSettings, string, error) {
	configFile := AppSettingsFileName
	if dir != "" {
		configFile = filepath.Join(dir, configFile)
	}

	// Get absolute path for display
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		absPath = configFile
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if file doesn't exist
			return DefaultAppSettings(), absPath, nil
		}
		return nil, absPath, err
	}

	settings := DefaultAppSettings()
	if err := yaml.Unmarshal(data, settings); err != nil {
		return nil, absPath, err
	}

	return settings, absPath, nil
}

// SaveAppSettings saves app settings to orchestrate.yaml in the specified directory.
func SaveAppSettings(dir string, settings *AppSettings) error {
	configFile := AppSettingsFileName
	if dir != "" {
		configFile = filepath.Join(dir, configFile)
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}

	// Add a header comment
	header := "# Orchestrate App Settings\n# This file is auto-generated. Edit carefully.\n\n"
	return os.WriteFile(configFile, []byte(header+string(data)), 0644)
}

// GetTerminalTypeOptions returns the available terminal type options
func GetTerminalTypeOptions() []TerminalType {
	return []TerminalType{TerminalITerm2, TerminalRegular}
}

// GetThemeOptions returns the available theme options
func GetThemeOptions() []string {
	return []string{"default", "dark", "light"}
}
