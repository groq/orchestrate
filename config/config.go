// Package config handles configuration loading for orchestrate.
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Command represents a custom command to run in an agent's worktree.
type Command struct {
	Command string `yaml:"command,omitempty"` // Shell command to run (empty = just open terminal)
	Title   string `yaml:"title,omitempty"`   // Custom title for the window
	Color   string `yaml:"color,omitempty"`   // Hex color for tab (e.g., "#ff0000")
}

// GetTitle returns the display title for the command.
func (c Command) GetTitle() string {
	if c.Title != "" {
		return c.Title
	}
	if c.Command == "" {
		return "terminal"
	}
	// Truncate long commands for title
	if len(c.Command) > 30 {
		return c.Command[:27] + "..."
	}
	return c.Command
}

// Worktree represents a single agent configuration with optional commands.
type Worktree struct {
	Agent    string    `yaml:"agent,omitempty"`    // Agent name (e.g., "claude", "codex")
	N        int       `yaml:"n,omitempty"`        // Multiplier for this agent (default 1)
	Commands []Command `yaml:"commands,omitempty"` // Commands to run in this agent's worktree
}

// GetN returns the multiplier for this worktree (defaults to 1 if not set).
func (w Worktree) GetN() int {
	if w.N <= 0 {
		return 1
	}
	return w.N
}

// HasCommands returns true if this agent has associated commands.
func (w Worktree) HasCommands() bool {
	return len(w.Commands) > 0
}

// IsValid returns true if this worktree has a valid configuration.
// A valid worktree must have an agent name.
func (w Worktree) IsValid() bool {
	return w.Agent != ""
}

// Preset is an ordered list of agent worktrees.
type Preset []Worktree

// Config represents the settings.orchestrate.yaml configuration file.
type Config struct {
	Default string            `yaml:"default"`
	Presets map[string]Preset `yaml:"presets"`
}

// SettingsFileName is the name of the settings file.
const SettingsFileName = "settings.yaml"

// LoadResult contains the loaded configuration and its path.
type LoadResult struct {
	Config *Config
	Path   string
}

// Load loads configuration from settings.orchestrate.yaml in the specified directory.
// The directory should be the orchestrate data directory.
// Returns LoadResult with nil Config if file doesn't exist.
func Load(dir string) LoadResult {
	configFile := SettingsFileName
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
		return LoadResult{Config: nil, Path: ""} // No config file, use defaults
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Printf("Warning: invalid %s", configFile)
		return LoadResult{Config: nil, Path: ""}
	}

	return LoadResult{Config: &config, Path: absPath}
}

// LoadFromBytes parses configuration from YAML bytes.
// This is useful for testing or loading config from non-file sources.
func LoadFromBytes(data []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetPreset retrieves a preset by name from the config.
// Returns the preset and true if found, empty Preset and false otherwise.
func (c *Config) GetPreset(name string) (Preset, bool) {
	if c == nil || c.Presets == nil {
		return Preset{}, false
	}
	preset, ok := c.Presets[name]
	return preset, ok
}

// GetDefaultPresetName returns the default preset name.
// Returns empty string if no default is set.
func (c *Config) GetDefaultPresetName() string {
	if c == nil {
		return ""
	}
	return c.Default
}

// SavePresetConfig saves the preset configuration to settings.yaml in the specified directory.
// The directory should be the orchestrate data directory.
func SavePresetConfig(dir string, config *Config) error {
	configFile := SettingsFileName
	if dir != "" {
		configFile = filepath.Join(dir, configFile)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with proper permissions (0644 = rw-r--r--)
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ParseHexColor parses a hex color string (e.g., "#ff8c00") into RGB values.
// Returns r, g, b values and true if valid, or 0, 0, 0 and false if invalid.
func ParseHexColor(hex string) (r, g, b int, ok bool) {
	if len(hex) == 0 {
		return 0, 0, 0, false
	}
	// Remove leading #
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	// Parse hex values
	var rv, gv, bv int
	_, err := sscanHex(hex[0:2], &rv)
	if err != nil {
		return 0, 0, 0, false
	}
	_, err = sscanHex(hex[2:4], &gv)
	if err != nil {
		return 0, 0, 0, false
	}
	_, err = sscanHex(hex[4:6], &bv)
	if err != nil {
		return 0, 0, 0, false
	}
	return rv, gv, bv, true
}

// sscanHex parses a hex string into an int.
func sscanHex(s string, v *int) (int, error) {
	var val int
	for _, c := range s {
		val *= 16
		switch {
		case c >= '0' && c <= '9':
			val += int(c - '0')
		case c >= 'a' && c <= 'f':
			val += int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			val += int(c-'A') + 10
		default:
			return 0, fmt.Errorf("invalid hex char: %c", c)
		}
	}
	*v = val
	return 1, nil
}
