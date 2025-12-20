package header

import (
	"testing"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	header := New(ctx)

	if header.ctx == nil {
		t.Error("ctx should not be nil")
	}
}

func TestNew_NilContext(t *testing.T) {
	header := New(nil)

	// Should not panic
	if header.ctx != nil {
		t.Error("ctx should be nil when passed nil")
	}
}

func TestModel_Height(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	header := New(ctx)

	if header.Height() != constants.HeaderHeight {
		t.Errorf("Height = %d, want %d", header.Height(), constants.HeaderHeight)
	}
}

func TestModel_View_NilContext(t *testing.T) {
	header := New(nil)

	view := header.View()
	if view != "" {
		t.Error("View should return empty string with nil context")
	}
}

func TestModel_View_Normal(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 40})
	header := New(ctx)

	view := header.View()
	if view == "" {
		t.Error("View should not be empty")
	}
}

func TestModel_UpdateProgramContext(t *testing.T) {
	header := New(nil)

	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	header.UpdateProgramContext(ctx)

	if header.ctx == nil {
		t.Error("ctx should not be nil after update")
	}
}

func TestModel_View_DifferentViews(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 40})

	views := []constants.ViewType{
		constants.WorktreesView,
		constants.SettingsView,
		constants.PresetsView,
	}

	for _, view := range views {
		ctx.SetView(view)
		header := New(ctx)

		rendered := header.View()
		if rendered == "" {
			t.Errorf("View should not be empty for view type %v", view)
		}
	}
}

