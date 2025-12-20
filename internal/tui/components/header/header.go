// Package header provides the header component for the TUI.
// Styled after gh-dash's tabs component.
package header

import (
	"fmt"

	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
	"orchestrate/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the header component.
type Model struct {
	ctx     *context.ProgramContext
	focused bool
}

// New creates a new header model.
func New(ctx *context.ProgramContext) Model {
	return Model{ctx: ctx}
}

// SetFocused sets the focus state of the header.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// Init initializes the header.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

// View renders the header.
// Layout: [Tabs...] [spacer] [Logo + version]
func (m Model) View() string {
	if m.ctx == nil {
		return ""
	}

	tabs := m.renderTabs()
	logo := m.renderLogo()

	tabsWidth := lipgloss.Width(tabs)
	logoWidth := lipgloss.Width(logo)
	spacerWidth := m.ctx.ScreenWidth - tabsWidth - logoWidth

	if spacerWidth < 0 {
		spacerWidth = 0
	}

	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	content := lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		tabs,
		spacer,
		logo,
	)

	return m.ctx.Styles.Tabs.TabsRow.
		Width(m.ctx.ScreenWidth).
		Height(constants.TabsContentHeight).
		MaxHeight(constants.TabsContentHeight).
		Render(content)
}

func (m Model) renderTabs() string {
	views := []constants.ViewType{
		constants.WorktreesView,
		constants.LaunchView,
		constants.SettingsView,
		constants.PresetsView,
	}

	var tabs []string
	for _, v := range views {
		style := m.ctx.Styles.Tabs.Tab
		name := fmt.Sprintf("%s %s", v.Icon(), v.String())

		if m.ctx.View == v {
			style = m.ctx.Styles.Tabs.ActiveTab
			if m.focused {
				name = fmt.Sprintf("> %s <", name)
			}
		}

		tabs = append(tabs, style.Render(name))
	}

	separator := m.ctx.Styles.Tabs.TabSeparator.Render(" | ")
	result := ""
	for i, tab := range tabs {
		if i > 0 {
			result += separator
		}
		result += tab
	}

	return result
}

func (m Model) renderLogo() string {
	orchStyle := lipgloss.NewStyle().
		Foreground(m.ctx.Theme.PrimaryText).
		Bold(true)

	byStyle := lipgloss.NewStyle().
		Foreground(m.ctx.Theme.FaintText)

	groqStyle := lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Bold(true)

	return lipgloss.NewStyle().
		Padding(0, 1, 0, 2).
		Height(1).
		Render(lipgloss.JoinHorizontal(lipgloss.Bottom,
			orchStyle.Render("Orchestrate"),
			byStyle.Render(" by "),
			groqStyle.Render("Groq"),
		))
}

// UpdateProgramContext updates the context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}

// Height returns the header height.
func (m Model) Height() int {
	return constants.HeaderHeight
}
