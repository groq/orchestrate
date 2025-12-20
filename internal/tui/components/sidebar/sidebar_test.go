package sidebar

import (
	"testing"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	if sidebar.ctx == nil {
		t.Error("ctx should not be nil")
	}
	if sidebar.IsOpen {
		t.Error("IsOpen should be false initially")
	}
}

func TestNew_NilContext(t *testing.T) {
	sidebar := New(nil)

	// Should not panic
	if sidebar.ctx != nil {
		t.Error("ctx should be nil when passed nil")
	}
}

func TestModel_Toggle(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	if sidebar.IsOpen {
		t.Error("Sidebar should start closed")
	}

	sidebar.Toggle()
	if !sidebar.IsOpen {
		t.Error("Sidebar should be open after toggle")
	}

	sidebar.Toggle()
	if sidebar.IsOpen {
		t.Error("Sidebar should be closed after second toggle")
	}
}

func TestModel_OpenClose(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	sidebar.Open()
	if !sidebar.IsOpen {
		t.Error("Sidebar should be open after Open()")
	}

	sidebar.Close()
	if sidebar.IsOpen {
		t.Error("Sidebar should be closed after Close()")
	}
}

func TestModel_SetContent(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	sidebar.SetContent("test content")
	if sidebar.content != "test content" {
		t.Errorf("content = %q, want %q", sidebar.content, "test content")
	}
}

func TestModel_SetTitle(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	sidebar.SetTitle("test title")
	if sidebar.title != "test title" {
		t.Errorf("title = %q, want %q", sidebar.title, "test title")
	}
}

func TestModel_Width(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	if sidebar.Width() != 0 {
		t.Errorf("Width when closed = %d, want 0", sidebar.Width())
	}

	sidebar.Open()
	if sidebar.Width() != constants.SidebarWidth {
		t.Errorf("Width when open = %d, want %d", sidebar.Width(), constants.SidebarWidth)
	}
}

func TestModel_GetContentWidth(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	width := sidebar.GetContentWidth()
	if width <= 0 {
		t.Errorf("GetContentWidth = %d, should be positive", width)
	}
	if width >= constants.SidebarWidth {
		t.Error("GetContentWidth should be less than SidebarWidth")
	}
}

func TestModel_View_Closed(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	sidebar := New(ctx)

	view := sidebar.View()
	if view != "" {
		t.Error("View should be empty when sidebar is closed")
	}
}

func TestModel_View_OpenEmpty(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 40})
	sidebar := New(ctx)
	sidebar.UpdateProgramContext(ctx)

	sidebar.Open()
	view := sidebar.View()
	if view == "" {
		t.Error("View should not be empty when sidebar is open")
	}
}

func TestModel_View_OpenWithContent(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 40})
	sidebar := New(ctx)
	sidebar.UpdateProgramContext(ctx)

	sidebar.Open()
	sidebar.SetContent("some content")
	view := sidebar.View()
	if view == "" {
		t.Error("View should not be empty when sidebar has content")
	}
}

func TestModel_UpdateProgramContext(t *testing.T) {
	sidebar := New(nil)

	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	ctx.UpdateWindowSize(tea.WindowSizeMsg{Width: 100, Height: 40})
	sidebar.UpdateProgramContext(ctx)

	if sidebar.ctx == nil {
		t.Error("ctx should not be nil after update")
	}
}

