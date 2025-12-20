// Package sidebar provides the sidebar component for the TUI.
package sidebar

import (
	"fmt"

	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
	"orchestrate/internal/tui/keys"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the sidebar component.
type Model struct {
	ctx        *context.ProgramContext
	IsOpen     bool
	title      string
	content    string
	viewport   viewport.Model
	emptyState string
}

// New creates a new sidebar model.
func New(ctx *context.ProgramContext) Model {
	return Model{
		ctx:        ctx,
		IsOpen:     false,
		emptyState: "Nothing selected...",
		viewport: viewport.Model{
			Width:  constants.SidebarWidth,
			Height: 0,
		},
	}
}

// Init initializes the sidebar.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Keys.Down):
			m.viewport.ScrollDown(1)
		case key.Matches(msg, keys.Keys.Up):
			m.viewport.ScrollUp(1)
		}

	case tea.WindowSizeMsg:
		m.updateDimensions()
	}

	return m, nil
}

// View renders the sidebar.
func (m Model) View() string {
	if !m.IsOpen || m.ctx == nil {
		return ""
	}

	height := m.ctx.MainContentHeight
	width := constants.SidebarWidth

	style := m.ctx.Styles.Sidebar.Root.
		Height(height).
		Width(width).
		MaxWidth(width)

	if m.content == "" {
		return style.
			Align(lipgloss.Center).
			Render(lipgloss.PlaceVertical(height, lipgloss.Center, m.emptyState))
	}

	// Title
	titleView := ""
	if m.title != "" {
		titleView = m.ctx.Styles.Sidebar.Title.Render(m.title) + "\n"
	}

	// Pager
	pagerView := m.ctx.Styles.Sidebar.PagerStyle.
		Render(fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100)))

	return style.Render(lipgloss.JoinVertical(
		lipgloss.Top,
		titleView,
		m.viewport.View(),
		pagerView,
	))
}

// SetContent sets the sidebar content.
func (m *Model) SetContent(content string) {
	m.content = content
	m.viewport.SetContent(content)
}

// SetTitle sets the sidebar title.
func (m *Model) SetTitle(title string) {
	m.title = title
}

// Toggle toggles the sidebar visibility.
func (m *Model) Toggle() {
	m.IsOpen = !m.IsOpen
}

// Open opens the sidebar.
func (m *Model) Open() {
	m.IsOpen = true
}

// Close closes the sidebar.
func (m *Model) Close() {
	m.IsOpen = false
}

// ScrollToTop scrolls to the top.
func (m *Model) ScrollToTop() {
	m.viewport.GotoTop()
}

// ScrollToBottom scrolls to the bottom.
func (m *Model) ScrollToBottom() {
	m.viewport.GotoBottom()
}

// GetContentWidth returns the content width.
func (m Model) GetContentWidth() int {
	if m.ctx == nil {
		return constants.SidebarWidth - 2
	}
	return constants.SidebarWidth - m.ctx.Styles.Sidebar.BorderWidth - m.ctx.Styles.Sidebar.ContentPad*2
}

// UpdateProgramContext updates the context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
	m.updateDimensions()
}

func (m *Model) updateDimensions() {
	if m.ctx == nil {
		return
	}
	m.viewport.Height = m.ctx.MainContentHeight - m.ctx.Styles.Sidebar.PagerHeight - 2
	m.viewport.Width = m.GetContentWidth()
}

// Width returns the sidebar width when open.
func (m Model) Width() int {
	if m.IsOpen {
		return constants.SidebarWidth
	}
	return 0
}

