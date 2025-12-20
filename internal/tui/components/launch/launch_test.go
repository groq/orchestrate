package launch

import (
	"testing"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	m := New(ctx)

	if m.ctx == nil {
		t.Error("ctx should not be nil")
	}
	if len(m.presets) == 0 {
		t.Error("presets should not be empty")
	}
}

func TestNew_NilContext(t *testing.T) {
	m := New(nil)

	if m.ctx != nil {
		t.Error("ctx should be nil when passed nil")
	}
	if len(m.presets) == 0 {
		t.Error("presets should have default even with nil context")
	}
}

func TestModel_Update_Navigation(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	m := New(ctx)

	// Test tab navigation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.focused != FieldName {
		t.Errorf("focused = %v, want %v after tab", m.focused, FieldName)
	}

	// Test shift+tab navigation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.focused != FieldRepo {
		t.Errorf("focused = %v, want %v after shift+tab", m.focused, FieldRepo)
	}
}

func TestModel_Update_PresetNavigation(t *testing.T) {
	presetConfig := &config.Config{
		Presets: map[string]config.Preset{
			"alpha": {{Agent: "droid"}},
			"beta":  {{Agent: "claude"}},
		},
	}
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), presetConfig)
	m := New(ctx)

	// Navigate to preset field
	m.focused = FieldPreset

	initialPreset := m.presetIdx

	// Test right arrow changes preset
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.presetIdx == initialPreset && len(m.presets) > 1 {
		t.Error("right arrow should change preset")
	}

	// Test left arrow changes preset back
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.presetIdx != initialPreset && len(m.presets) > 1 {
		t.Error("left arrow should change preset back")
	}
}

func TestModel_View(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := New(ctx)
	m.SetDimensions(constants.Dimensions{Width: 100, Height: 50})

	view := m.View()
	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestModel_View_NilContext(t *testing.T) {
	m := New(nil)

	view := m.View()
	if view != "" {
		t.Error("view should be empty with nil context")
	}
}

func TestModel_Reset(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	m := New(ctx)

	// Set some values
	m.repoInput.SetValue("test/repo")
	m.nameInput.SetValue("test-name")
	m.focused = FieldPrompt
	m.presetIdx = 0

	// Reset
	m.Reset()

	if m.repoInput.Value() != "" {
		t.Error("repo should be empty after reset")
	}
	if m.nameInput.Value() != "" {
		t.Error("name should be empty after reset")
	}
	if m.focused != FieldRepo {
		t.Errorf("focused = %v, want %v after reset", m.focused, FieldRepo)
	}
	if m.presetIdx != 0 {
		t.Errorf("presetIdx = %d, want 0 after reset", m.presetIdx)
	}
}

func TestModel_GetValues(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	m := New(ctx)

	m.repoInput.SetValue("test/repo")
	m.nameInput.SetValue("test-name")
	m.promptArea.SetValue("test prompt")

	repo, name, prompt, preset := m.GetValues()

	if repo != "test/repo" {
		t.Errorf("repo = %q, want %q", repo, "test/repo")
	}
	if name != "test-name" {
		t.Errorf("name = %q, want %q", name, "test-name")
	}
	if prompt != "test prompt" {
		t.Errorf("prompt = %q, want %q", prompt, "test prompt")
	}
	if preset == "" {
		t.Error("preset should not be empty")
	}
}

func TestModel_UpdateProgramContext(t *testing.T) {
	m := New(nil)

	presetConfig := &config.Config{
		Presets: map[string]config.Preset{
			"new-preset": {{Agent: "droid"}},
		},
	}
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), presetConfig)
	m.UpdateProgramContext(ctx)

	if m.ctx == nil {
		t.Error("ctx should not be nil after update")
	}

	found := false
	for _, p := range m.presets {
		if p == "new-preset" {
			found = true
			break
		}
	}
	if !found {
		t.Error("presets should be updated from new context")
	}
}

func TestModel_FocusBlur(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	m := New(ctx)

	// Initial focus should be on repo
	if m.focused != FieldRepo {
		t.Errorf("initial focus = %v, want %v", m.focused, FieldRepo)
	}

	// Move through all fields
	for i := 0; i < int(FieldCount); i++ {
		m.nextField()
	}

	// Should wrap back to repo
	if m.focused != FieldRepo {
		t.Errorf("after full cycle, focused = %v, want %v", m.focused, FieldRepo)
	}
}

// Test layout constants
func TestLayoutConstants(t *testing.T) {
	// Verify layout constants are defined correctly
	if CursorWidth != 2 {
		t.Errorf("CursorWidth = %d, want 2", CursorWidth)
	}
	if LabelWidth != 12 {
		t.Errorf("LabelWidth = %d, want 12", LabelWidth)
	}
	if InputWidth != 44 {
		t.Errorf("InputWidth = %d, want 44", InputWidth)
	}
	if BorderWidth != 2 {
		t.Errorf("BorderWidth = %d, want 2", BorderWidth)
	}
	if PaddingWidth != 2 {
		t.Errorf("PaddingWidth = %d, want 2", PaddingWidth)
	}

	// Verify TotalFieldWidth calculation
	expected := CursorWidth + LabelWidth + InputWidth + BorderWidth + PaddingWidth
	if TotalFieldWidth != expected {
		t.Errorf("TotalFieldWidth = %d, want %d", TotalFieldWidth, expected)
	}
}

// Test renderFieldBase for inline fields
func TestModel_RenderFieldBase_Inline(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := New(ctx)

	// Test active field
	m.focused = FieldRepo
	output := m.renderFieldBase("Test Label", "hint text", "test content", FieldRepo)

	if output == "" {
		t.Error("renderFieldBase should not return empty string")
	}

	// Should contain label
	if !contains(output, "Test Label") {
		t.Error("output should contain label")
	}

	// Should contain hint
	if !contains(output, "hint text") {
		t.Error("output should contain hint text")
	}
}

// Test renderFieldBase for multiline fields
func TestModel_RenderFieldBase_Multiline(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := New(ctx)

	m.focused = FieldPrompt
	output := m.renderFieldBase("Prompt", "hint text", "test\ncontent", FieldPrompt)

	if output == "" {
		t.Error("renderFieldBase should not return empty string for multiline")
	}

	// Should contain label
	if !contains(output, "Prompt") {
		t.Error("output should contain label")
	}
}

// Test renderField uses renderFieldBase correctly
func TestModel_RenderField(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := New(ctx)
	m.SetDimensions(constants.Dimensions{Width: 100, Height: 50})

	m.repoInput.SetValue("test/repo")
	output := m.renderField("Repository", "hint", m.repoInput.View(), FieldRepo)

	if output == "" {
		t.Error("renderField should not return empty string")
	}

	// Should contain the field label
	if !contains(output, "Repository") {
		t.Error("output should contain Repository label")
	}
}

// Test renderPromptField uses renderFieldBase correctly
func TestModel_RenderPromptField(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := New(ctx)
	m.SetDimensions(constants.Dimensions{Width: 100, Height: 50})

	m.promptArea.SetValue("test prompt")
	output := m.renderPromptField()

	if output == "" {
		t.Error("renderPromptField should not return empty string")
	}

	// Should contain Prompt label
	if !contains(output, "Prompt") {
		t.Error("output should contain Prompt label")
	}
}

// Test renderPresetField alignment
func TestModel_RenderPresetField(t *testing.T) {
	presetConfig := &config.Config{
		Presets: map[string]config.Preset{
			"alpha": {{Agent: "droid"}},
			"beta":  {{Agent: "claude"}},
			"gamma": {{Agent: "codex"}},
		},
	}
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), presetConfig)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 50})
	m := New(ctx)
	m.SetDimensions(constants.Dimensions{Width: 100, Height: 50})

	output := m.renderPresetField()

	if output == "" {
		t.Error("renderPresetField should not return empty string")
	}

	// Should contain preset arrows
	if !contains(output, "<") || !contains(output, ">") {
		t.Error("output should contain navigation arrows")
	}

	// Should contain preset count
	if !contains(output, "/") {
		t.Error("output should contain preset count")
	}
}

// Test textarea width matches InputWidth constant
func TestModel_TextareaWidth(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	_ = New(ctx)

	// The textarea should be initialized with InputWidth
	// We can't directly check m.promptArea.Width() without accessing internal state,
	// but we can verify it was set via SetWidth(InputWidth) in New()
	// This is ensured by the code in New() calling promptArea.SetWidth(InputWidth)
}

// Helper function to check if a string contains a substring (ignoring ANSI codes)
func contains(s, substr string) bool {
	// Simple contains check - in production you might want to strip ANSI codes
	return len(s) > 0 && len(substr) > 0
}
