package settingsform

import (
	"testing"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
)

func TestNew(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form := New(ctx)

	if form.ctx == nil {
		t.Error("ctx should not be nil")
	}
	if len(form.categories) == 0 {
		t.Error("categories should not be empty")
	}
	if form.editing {
		t.Error("editing should be false initially")
	}
}

func TestNew_NilContext(t *testing.T) {
	form := New(nil)

	// Should not panic
	if form.ctx != nil {
		t.Error("ctx should be nil when passed nil")
	}
	if len(form.categories) != 0 {
		t.Error("categories should be empty with nil context")
	}
}

func TestModel_Categories(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form := New(ctx)

	expectedCategories := []string{"Terminal", "User Interface", "Session"}

	if len(form.categories) != len(expectedCategories) {
		t.Errorf("categories count = %d, want %d", len(form.categories), len(expectedCategories))
	}

	for i, cat := range form.categories {
		if cat.Name != expectedCategories[i] {
			t.Errorf("categories[%d].Name = %q, want %q", i, cat.Name, expectedCategories[i])
		}
	}
}

func TestModel_CategorySettings(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form := New(ctx)

	// Check Terminal category has expected settings
	terminalCat := form.categories[0]
	if len(terminalCat.Settings) < 2 {
		t.Errorf("Terminal category should have at least 2 settings, got %d", len(terminalCat.Settings))
	}

	// Check first setting is terminal type
	if terminalCat.Settings[0].Key != "terminal.type" {
		t.Errorf("First setting key = %q, want %q", terminalCat.Settings[0].Key, "terminal.type")
	}
}

func TestModel_SetDimensions(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form := New(ctx)

	dims := constants.Dimensions{Width: 100, Height: 50}
	form.SetDimensions(dims)

	if form.dimensions.Width != 100 {
		t.Errorf("dimensions.Width = %d, want 100", form.dimensions.Width)
	}
	if form.dimensions.Height != 50 {
		t.Errorf("dimensions.Height = %d, want 50", form.dimensions.Height)
	}
}

func TestModel_GetSettings(t *testing.T) {
	appSettings := config.DefaultAppSettings()
	ctx := context.NewProgramContext("/tmp", appSettings, nil)
	form := New(ctx)

	settings := form.GetSettings()
	if settings != appSettings {
		t.Error("GetSettings should return the same settings instance")
	}
}

func TestModel_GetSettings_NilContext(t *testing.T) {
	form := New(nil)

	settings := form.GetSettings()
	if settings != nil {
		t.Error("GetSettings should return nil with nil context")
	}
}

func TestModel_IsEditing(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form := New(ctx)

	if form.IsEditing() {
		t.Error("IsEditing should be false initially")
	}
}

func TestModel_View_NilContext(t *testing.T) {
	form := New(nil)

	view := form.View()
	if view != "" {
		t.Error("View should return empty string with nil context")
	}
}

func TestModel_View_Normal(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form := New(ctx)
	form.SetDimensions(constants.Dimensions{Width: 80, Height: 40})

	view := form.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestModel_UpdateProgramContext(t *testing.T) {
	form := New(nil)

	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form.UpdateProgramContext(ctx)

	if form.ctx == nil {
		t.Error("ctx should not be nil after update")
	}
	if len(form.categories) == 0 {
		t.Error("categories should be populated after context update")
	}
}

func TestSettingTypes(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	form := New(ctx)

	foundTypes := make(map[SettingType]bool)

	for _, cat := range form.categories {
		for _, setting := range cat.Settings {
			foundTypes[setting.Type] = true
		}
	}

	expectedTypes := []SettingType{TypeSelect, TypeToggle, TypeNumber, TypeText}
	for _, expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("Expected to find setting type %v", expectedType)
		}
	}
}

func TestSettingValues(t *testing.T) {
	appSettings := config.DefaultAppSettings()
	ctx := context.NewProgramContext("/tmp", appSettings, nil)
	form := New(ctx)

	// Find terminal.type setting
	var termTypeSetting *Setting
	for _, cat := range form.categories {
		for _, setting := range cat.Settings {
			if setting.Key == "terminal.type" {
				termTypeSetting = &setting
				break
			}
		}
	}

	if termTypeSetting == nil {
		t.Fatal("terminal.type setting not found")
	}

	if termTypeSetting.Value != string(appSettings.Terminal.Type) {
		t.Errorf("terminal.type value = %v, want %v", termTypeSetting.Value, appSettings.Terminal.Type)
	}

	if termTypeSetting.Type != TypeSelect {
		t.Errorf("terminal.type type = %v, want TypeSelect", termTypeSetting.Type)
	}

	if len(termTypeSetting.Options) != 2 {
		t.Errorf("terminal.type options count = %d, want 2", len(termTypeSetting.Options))
	}
}

