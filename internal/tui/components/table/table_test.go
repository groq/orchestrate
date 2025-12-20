package table

import (
	"testing"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
)

func TestNew(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	columns := []Column{
		{Title: "Name", Width: 20},
		{Title: "Value", Grow: true},
	}

	table := New(ctx, columns)

	if table.ctx == nil {
		t.Error("ctx should not be nil")
	}
	if len(table.columns) != 2 {
		t.Errorf("columns = %d, want 2", len(table.columns))
	}
	if table.cursor != 0 {
		t.Errorf("cursor = %d, want 0", table.cursor)
	}
}

func TestModel_SetRows(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	table := New(ctx, []Column{{Title: "Test"}})

	rows := []Row{
		{"Row 1"},
		{"Row 2"},
		{"Row 3"},
	}
	table.SetRows(rows)

	if table.NumRows() != 3 {
		t.Errorf("NumRows = %d, want 3", table.NumRows())
	}
}

func TestModel_CursorNavigation(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	table := New(ctx, []Column{{Title: "Test"}})
	table.SetDimensions(constants.Dimensions{Width: 80, Height: 20})
	table.SetRows([]Row{{"1"}, {"2"}, {"3"}})

	// Test cursor down
	table.CursorDown()
	if table.Cursor() != 1 {
		t.Errorf("Cursor = %d, want 1", table.Cursor())
	}

	// Test cursor down again
	table.CursorDown()
	if table.Cursor() != 2 {
		t.Errorf("Cursor = %d, want 2", table.Cursor())
	}

	// Test cursor at end
	table.CursorDown()
	if table.Cursor() != 2 {
		t.Errorf("Cursor should stay at 2, got %d", table.Cursor())
	}

	// Test cursor up
	table.CursorUp()
	if table.Cursor() != 1 {
		t.Errorf("Cursor = %d, want 1", table.Cursor())
	}

	// Test cursor first
	table.CursorFirst()
	if table.Cursor() != 0 {
		t.Errorf("Cursor = %d, want 0", table.Cursor())
	}

	// Test cursor last
	table.CursorLast()
	if table.Cursor() != 2 {
		t.Errorf("Cursor = %d, want 2", table.Cursor())
	}
}

func TestModel_SelectedRow(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	table := New(ctx, []Column{{Title: "Test"}})
	table.SetRows([]Row{{"Row 1"}, {"Row 2"}})

	selected := table.SelectedRow()
	if selected == nil || selected[0] != "Row 1" {
		t.Errorf("SelectedRow = %v, want [Row 1]", selected)
	}

	table.CursorDown()
	selected = table.SelectedRow()
	if selected == nil || selected[0] != "Row 2" {
		t.Errorf("SelectedRow = %v, want [Row 2]", selected)
	}
}

func TestModel_Loading(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	table := New(ctx, []Column{{Title: "Test"}})

	if table.IsLoading() {
		t.Error("Table should not be loading initially")
	}

	table.SetLoading(true, "Loading...")
	if !table.IsLoading() {
		t.Error("Table should be loading after SetLoading(true)")
	}

	table.SetLoading(false, "")
	if table.IsLoading() {
		t.Error("Table should not be loading after SetLoading(false)")
	}
}

func TestModel_EmptyRows(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	table := New(ctx, []Column{{Title: "Test"}})
	table.SetDimensions(constants.Dimensions{Width: 80, Height: 20})

	// Empty table
	if table.NumRows() != 0 {
		t.Errorf("NumRows = %d, want 0", table.NumRows())
	}

	// SelectedRow on empty table
	if table.SelectedRow() != nil {
		t.Error("SelectedRow should be nil on empty table")
	}
}

func TestModel_SetDimensions(t *testing.T) {
	ctx := context.NewProgramContext("/tmp", config.DefaultAppSettings(), nil)
	table := New(ctx, []Column{{Title: "Test"}})

	dims := constants.Dimensions{Width: 100, Height: 30}
	table.SetDimensions(dims)

	if table.dimensions.Width != 100 {
		t.Errorf("dimensions.Width = %d, want 100", table.dimensions.Width)
	}
	if table.dimensions.Height != 30 {
		t.Errorf("dimensions.Height = %d, want 30", table.dimensions.Height)
	}
}

