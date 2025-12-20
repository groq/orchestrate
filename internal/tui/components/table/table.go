// Package table provides a reusable table component for the TUI.
package table

import (
	"strings"

	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Column represents a table column.
type Column struct {
	Title string
	Width int
	Grow  bool
}

// Row represents a table row.
type Row []string

// Model represents the table component.
type Model struct {
	ctx            *context.ProgramContext
	columns        []Column
	rows           []Row
	cursor         int
	offset         int
	isLoading      bool
	loadingMessage string
	emptyMessage   string
	spinner        spinner.Model
	dimensions     constants.Dimensions
}

// New creates a new table model.
func New(ctx *context.ProgramContext, columns []Column) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return Model{
		ctx:            ctx,
		columns:        columns,
		rows:           []Row{},
		cursor:         0,
		offset:         0,
		isLoading:      false,
		loadingMessage: "Loading...",
		emptyMessage:   "No items",
		spinner:        s,
	}
}

// Init initializes the table.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if m.isLoading {
		m.spinner, cmd = m.spinner.Update(msg)
	}

	return m, cmd
}

// View renders the table.
func (m Model) View() string {
	if m.ctx == nil {
		return ""
	}

	header := m.renderHeader()
	body := m.renderBody()

	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func (m Model) renderHeader() string {
	cells := make([]string, 0, len(m.columns))
	totalWidth := 0
	growCount := 0

	// First pass: calculate fixed widths
	for _, col := range m.columns {
		if col.Grow {
			growCount++
		} else {
			totalWidth += col.Width
		}
	}

	// Calculate grow width
	remainingWidth := m.dimensions.Width - totalWidth
	growWidth := 0
	if growCount > 0 {
		growWidth = remainingWidth / growCount
	}

	// Second pass: render columns
	for _, col := range m.columns {
		width := col.Width
		if col.Grow {
			width = growWidth
		}

		cell := m.ctx.Styles.Table.TitleCellStyle.
			Width(width).
			MaxWidth(width).
			Render(col.Title)
		cells = append(cells, cell)
	}

	return m.ctx.Styles.Table.HeaderStyle.
		Width(m.dimensions.Width).
		Height(constants.TableHeaderHeight).
		Render(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
}

func (m Model) renderBody() string {
	bodyStyle := lipgloss.NewStyle().
		Height(m.dimensions.Height - constants.TableHeaderHeight).
		Width(m.dimensions.Width)

	if m.isLoading {
		return lipgloss.Place(
			m.dimensions.Width,
			m.dimensions.Height-constants.TableHeaderHeight,
			lipgloss.Center,
			lipgloss.Center,
			m.spinner.View()+" "+m.loadingMessage,
		)
	}

	if len(m.rows) == 0 {
		return bodyStyle.Render(
			lipgloss.Place(
				m.dimensions.Width,
				m.dimensions.Height-constants.TableHeaderHeight,
				lipgloss.Center,
				lipgloss.Center,
				m.ctx.Styles.Table.EmptyState.Render(m.emptyMessage),
			),
		)
	}

	// Calculate visible rows
	visibleHeight := m.dimensions.Height - constants.TableHeaderHeight
	maxVisible := visibleHeight

	var renderedRows []string
	for i := m.offset; i < len(m.rows) && i-m.offset < maxVisible; i++ {
		renderedRows = append(renderedRows, m.renderRow(i))
	}

	return bodyStyle.Render(strings.Join(renderedRows, "\n"))
}

func (m Model) renderRow(index int) string {
	if index >= len(m.rows) {
		return ""
	}

	row := m.rows[index]
	cells := make([]string, 0, len(m.columns))

	// Calculate column widths (same as header)
	totalWidth := 0
	growCount := 0
	for _, col := range m.columns {
		if col.Grow {
			growCount++
		} else {
			totalWidth += col.Width
		}
	}
	remainingWidth := m.dimensions.Width - totalWidth
	growWidth := 0
	if growCount > 0 {
		growWidth = remainingWidth / growCount
	}

	isSelected := index == m.cursor

	for i, col := range m.columns {
		width := col.Width
		if col.Grow {
			width = growWidth
		}

		content := ""
		if i < len(row) {
			content = row[i]
		}

		style := m.ctx.Styles.Table.CellStyle
		if isSelected {
			style = m.ctx.Styles.Table.SelectedCellStyle
		}

		cell := style.
			Width(width).
			MaxWidth(width).
			Render(content)
		cells = append(cells, cell)
	}

	rowStyle := m.ctx.Styles.Table.RowStyle
	return rowStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, cells...))
}

// SetRows sets the table rows.
func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	if m.cursor >= len(rows) {
		m.cursor = max(0, len(rows)-1)
	}
}

// SetDimensions sets the table dimensions.
func (m *Model) SetDimensions(dimensions constants.Dimensions) {
	m.dimensions = dimensions
}

// SetLoading sets the loading state.
func (m *Model) SetLoading(loading bool, message string) {
	m.isLoading = loading
	if message != "" {
		m.loadingMessage = message
	}
}

// SetEmptyMessage sets the empty state message.
func (m *Model) SetEmptyMessage(message string) {
	m.emptyMessage = message
}

// CursorUp moves the cursor up.
func (m *Model) CursorUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
	}
}

// CursorDown moves the cursor down.
func (m *Model) CursorDown() {
	if m.cursor < len(m.rows)-1 {
		m.cursor++
		visibleHeight := m.dimensions.Height - constants.TableHeaderHeight
		if m.cursor >= m.offset+visibleHeight {
			m.offset = m.cursor - visibleHeight + 1
		}
	}
}

// CursorFirst moves cursor to first row.
func (m *Model) CursorFirst() {
	m.cursor = 0
	m.offset = 0
}

// CursorLast moves cursor to last row.
func (m *Model) CursorLast() {
	m.cursor = max(0, len(m.rows)-1)
	visibleHeight := m.dimensions.Height - constants.TableHeaderHeight
	m.offset = max(0, m.cursor-visibleHeight+1)
}

// Cursor returns the current cursor position.
func (m Model) Cursor() int {
	return m.cursor
}

// SelectedRow returns the currently selected row.
func (m Model) SelectedRow() Row {
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		return m.rows[m.cursor]
	}
	return nil
}

// NumRows returns the number of rows.
func (m Model) NumRows() int {
	return len(m.rows)
}

// IsLoading returns the loading state.
func (m Model) IsLoading() bool {
	return m.isLoading
}

// StartSpinner returns the spinner tick command.
func (m Model) StartSpinner() tea.Cmd {
	return m.spinner.Tick
}

// UpdateProgramContext updates the context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
	if ctx != nil {
		m.spinner.Style = lipgloss.NewStyle().Foreground(ctx.Theme.SecondaryText)
	}
}

