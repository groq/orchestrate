package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromBytes_BasicConfig(t *testing.T) {
	yaml := `
default: standard
presets:
  standard:
    - agent: claude
    - agent: codex
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	if cfg.Default != "standard" {
		t.Errorf("Default = %q, want 'standard'", cfg.Default)
	}

	preset, ok := cfg.GetPreset("standard")
	if !ok {
		t.Fatal("Preset 'standard' not found")
	}

	if len(preset) != 2 {
		t.Fatalf("len(preset) = %d, want 2", len(preset))
	}

	if preset[0].Agent != "claude" {
		t.Errorf("preset[0].Agent = %q, want 'claude'", preset[0].Agent)
	}
	if preset[1].Agent != "codex" {
		t.Errorf("preset[1].Agent = %q, want 'codex'", preset[1].Agent)
	}
}

func TestLoadFromBytes_AgentWithCommands(t *testing.T) {
	yaml := `
presets:
  dev:
    - agent: claude
      commands:
        - command: "npm run dev"
          title: "Dev Server"
          color: "#00ff00"
        - command: "npm test"
          title: "Tests"
    - agent: codex
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	preset, ok := cfg.GetPreset("dev")
	if !ok {
		t.Fatal("Preset 'dev' not found")
	}

	if len(preset) != 2 {
		t.Fatalf("len(preset) = %d, want 2", len(preset))
	}

	// First agent has commands
	if preset[0].Agent != "claude" {
		t.Errorf("preset[0].Agent = %q, want 'claude'", preset[0].Agent)
	}
	if !preset[0].HasCommands() {
		t.Error("preset[0] should have commands")
	}
	if len(preset[0].Commands) != 2 {
		t.Fatalf("len(preset[0].Commands) = %d, want 2", len(preset[0].Commands))
	}

	// Check first command
	cmd1 := preset[0].Commands[0]
	if cmd1.Command != "npm run dev" {
		t.Errorf("cmd1.Command = %q, want 'npm run dev'", cmd1.Command)
	}
	if cmd1.Title != "Dev Server" {
		t.Errorf("cmd1.Title = %q, want 'Dev Server'", cmd1.Title)
	}
	if cmd1.Color != "#00ff00" {
		t.Errorf("cmd1.Color = %q, want '#00ff00'", cmd1.Color)
	}

	// Check second command
	cmd2 := preset[0].Commands[1]
	if cmd2.Command != "npm test" {
		t.Errorf("cmd2.Command = %q, want 'npm test'", cmd2.Command)
	}

	// Second agent has no commands
	if preset[1].Agent != "codex" {
		t.Errorf("preset[1].Agent = %q, want 'codex'", preset[1].Agent)
	}
	if preset[1].HasCommands() {
		t.Error("preset[1] should not have commands")
	}
}

func TestLoadFromBytes_EmptyCommand(t *testing.T) {
	yaml := `
presets:
  terminal:
    - agent: claude
      commands:
        - command: ""
          title: "Extra Terminal"
        - title: "Another Terminal"
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	preset, _ := cfg.GetPreset("terminal")
	if len(preset[0].Commands) != 2 {
		t.Fatalf("len(commands) = %d, want 2", len(preset[0].Commands))
	}

	// First command is explicitly empty
	if preset[0].Commands[0].Command != "" {
		t.Error("First command should be empty")
	}
	if preset[0].Commands[0].Title != "Extra Terminal" {
		t.Errorf("Title = %q, want 'Extra Terminal'", preset[0].Commands[0].Title)
	}

	// Second command has no command field at all
	if preset[0].Commands[1].Command != "" {
		t.Error("Second command should be empty")
	}
}

func TestLoadFromBytes_OrderPreservation(t *testing.T) {
	yaml := `
presets:
  ordered:
    - agent: first
    - agent: second
    - agent: third
    - agent: fourth
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	preset, _ := cfg.GetPreset("ordered")
	expectedOrder := []string{"first", "second", "third", "fourth"}

	for i, w := range preset {
		if w.Agent != expectedOrder[i] {
			t.Errorf("preset[%d].Agent = %q, want %q", i, w.Agent, expectedOrder[i])
		}
	}
}

func TestLoadFromBytes_CommandOrderPreservation(t *testing.T) {
	yaml := `
presets:
  test:
    - agent: claude
      commands:
        - command: "first"
        - command: "second"
        - command: "third"
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	preset, _ := cfg.GetPreset("test")
	commands := preset[0].Commands

	expectedOrder := []string{"first", "second", "third"}
	for i, cmd := range commands {
		if cmd.Command != expectedOrder[i] {
			t.Errorf("commands[%d].Command = %q, want %q", i, cmd.Command, expectedOrder[i])
		}
	}
}

func TestLoadFromBytes_EmptyPreset(t *testing.T) {
	yaml := `
presets:
  empty: []
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	preset, _ := cfg.GetPreset("empty")
	if len(preset) != 0 {
		t.Errorf("len(preset) = %d, want 0", len(preset))
	}
}

func TestLoadFromBytes_InvalidYAML(t *testing.T) {
	yaml := `{{{invalid`
	_, err := LoadFromBytes([]byte(yaml))
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestLoad_FromFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configContent := `
default: test
presets:
  test:
    - agent: claude
      commands:
        - command: "echo hello"
          title: "Hello"
`
	configPath := filepath.Join(tmpDir, ".orchestrate.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	result := Load(tmpDir)
	if result.Config == nil {
		t.Fatal("Expected non-nil config")
	}
	if result.Config.Default != "test" {
		t.Errorf("Default = %q, want 'test'", result.Config.Default)
	}

	preset, ok := result.Config.GetPreset("test")
	if !ok {
		t.Fatal("Preset 'test' not found")
	}

	if len(preset) != 1 {
		t.Fatalf("len(preset) = %d, want 1", len(preset))
	}
	if !preset[0].HasCommands() {
		t.Error("Should have commands")
	}
}

func TestLoad_NoConfigFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "config_test_empty")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	result := Load(tmpDir)
	if result.Config != nil {
		t.Error("Expected nil config when file doesn't exist")
	}
}

func TestWindow_HasCommands(t *testing.T) {
	tests := []struct {
		name   string
		window Window
		want   bool
	}{
		{"no commands", Window{Agent: "claude"}, false},
		{"empty commands", Window{Agent: "claude", Commands: []Command{}}, false},
		{"with commands", Window{Agent: "claude", Commands: []Command{{Command: "test"}}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.window.HasCommands(); got != tt.want {
				t.Errorf("HasCommands() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommand_GetTitle(t *testing.T) {
	tests := []struct {
		name string
		cmd  Command
		want string
	}{
		{"explicit title", Command{Title: "My Title", Command: "test"}, "My Title"},
		{"command fallback short", Command{Command: "npm run"}, "npm run"},
		{"command fallback long", Command{Command: "npm run very-long-command-name-here"}, "npm run very-long-command-n..."},
		{"empty command", Command{}, "terminal"},
		{"empty command with title", Command{Title: "Shell"}, "Shell"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cmd.GetTitle(); got != tt.want {
				t.Errorf("GetTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfig_GetPreset(t *testing.T) {
	cfg := &Config{
		Default: "standard",
		Presets: map[string]Preset{
			"standard": {{Agent: "claude"}},
		},
	}

	tests := []struct {
		name       string
		config     *Config
		presetName string
		wantOk     bool
	}{
		{"existing preset", cfg, "standard", true},
		{"non-existing preset", cfg, "nonexistent", false},
		{"nil config", nil, "standard", false},
		{"config with nil presets", &Config{}, "standard", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := tt.config.GetPreset(tt.presetName)
			if ok != tt.wantOk {
				t.Errorf("GetPreset() ok = %v, want %v", ok, tt.wantOk)
			}
		})
	}
}

func TestConfig_GetDefaultPresetName(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   string
	}{
		{"config with default", &Config{Default: "standard"}, "standard"},
		{"config without default", &Config{}, ""},
		{"nil config", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetDefaultPresetName(); got != tt.want {
				t.Errorf("GetDefaultPresetName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		name   string
		hex    string
		wantR  int
		wantG  int
		wantB  int
		wantOk bool
	}{
		{"valid with hash", "#ff8c00", 255, 140, 0, true},
		{"valid without hash", "ff8c00", 255, 140, 0, true},
		{"valid green", "#00ff00", 0, 255, 0, true},
		{"valid blue", "#0000ff", 0, 0, 255, true},
		{"valid black", "#000000", 0, 0, 0, true},
		{"valid white", "#ffffff", 255, 255, 255, true},
		{"valid uppercase", "#AABBCC", 170, 187, 204, true},
		{"valid mixed case", "#AaBbCc", 170, 187, 204, true},
		{"too short", "#fff", 0, 0, 0, false},
		{"too long", "#fffffff", 0, 0, 0, false},
		{"invalid chars", "#gggggg", 0, 0, 0, false},
		{"empty", "", 0, 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b, ok := ParseHexColor(tt.hex)
			if ok != tt.wantOk {
				t.Errorf("ParseHexColor(%q) ok = %v, want %v", tt.hex, ok, tt.wantOk)
			}
			if tt.wantOk {
				if r != tt.wantR || g != tt.wantG || b != tt.wantB {
					t.Errorf("ParseHexColor(%q) = (%d, %d, %d), want (%d, %d, %d)",
						tt.hex, r, g, b, tt.wantR, tt.wantG, tt.wantB)
				}
			}
		})
	}
}

func TestLoadFromBytes_ComplexConfig(t *testing.T) {
	yaml := `
default: fullstack

presets:
  fullstack:
    - agent: claude
      commands:
        - command: "npm run dev"
          title: "Frontend Dev"
          color: "#00ccff"
        - command: "npm run test:watch"
          title: "Tests"
          color: "#ffff00"
    - agent: codex
      commands:
        - command: "cargo run"
          title: "Backend"
          color: "#ff6600"

  simple:
    - agent: droid
    - agent: claude
    - agent: codex
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	if cfg.Default != "fullstack" {
		t.Errorf("Default = %q, want 'fullstack'", cfg.Default)
	}

	// Test fullstack preset
	fullstack, ok := cfg.GetPreset("fullstack")
	if !ok {
		t.Fatal("Preset 'fullstack' not found")
	}

	if len(fullstack) != 2 {
		t.Fatalf("fullstack len = %d, want 2", len(fullstack))
	}

	// First agent (claude) has 2 commands
	if fullstack[0].Agent != "claude" {
		t.Error("fullstack[0] should be claude")
	}
	if len(fullstack[0].Commands) != 2 {
		t.Errorf("claude commands = %d, want 2", len(fullstack[0].Commands))
	}

	// Second agent (codex) has 1 command
	if fullstack[1].Agent != "codex" {
		t.Error("fullstack[1] should be codex")
	}
	if len(fullstack[1].Commands) != 1 {
		t.Errorf("codex commands = %d, want 1", len(fullstack[1].Commands))
	}

	// Test simple preset
	simple, ok := cfg.GetPreset("simple")
	if !ok {
		t.Fatal("Preset 'simple' not found")
	}

	if len(simple) != 3 {
		t.Fatalf("simple len = %d, want 3", len(simple))
	}

	// None should have commands
	for i, w := range simple {
		if w.HasCommands() {
			t.Errorf("simple[%d] should not have commands", i)
		}
	}
}

func TestLoadFromBytes_MultipleAgentsSameType(t *testing.T) {
	yaml := `
presets:
  heavy:
    - agent: claude
      commands:
        - command: "task1"
    - agent: claude
      commands:
        - command: "task2"
    - agent: claude
`
	cfg, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("LoadFromBytes failed: %v", err)
	}

	preset, _ := cfg.GetPreset("heavy")

	if len(preset) != 3 {
		t.Fatalf("len(preset) = %d, want 3", len(preset))
	}

	// All should be claude
	for i, w := range preset {
		if w.Agent != "claude" {
			t.Errorf("preset[%d].Agent = %q, want 'claude'", i, w.Agent)
		}
	}

	// First two have commands, third doesn't
	if len(preset[0].Commands) != 1 || preset[0].Commands[0].Command != "task1" {
		t.Error("First claude should have task1")
	}
	if len(preset[1].Commands) != 1 || preset[1].Commands[0].Command != "task2" {
		t.Error("Second claude should have task2")
	}
	if preset[2].HasCommands() {
		t.Error("Third claude should have no commands")
	}
}
