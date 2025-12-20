package context

import (
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/theme"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

// Styles contains all the computed styles for the TUI.
// Modeled after gh-dash's style system.
type Styles struct {
	// Common styles
	Common CommonStyles

	// Component-specific styles
	Header   HeaderStyles
	Tabs     TabStyles
	Sidebar  SidebarStyles
	Table    TableStyles
	Settings SettingsStyles
	Help     HelpStyles

	// Status colors
	Status StatusStyles
}

// CommonStyles contains frequently used styles.
type CommonStyles struct {
	MainTextStyle   lipgloss.Style
	FaintTextStyle  lipgloss.Style
	AccentTextStyle lipgloss.Style
	ErrorStyle      lipgloss.Style
	SuccessStyle    lipgloss.Style
	WarningStyle    lipgloss.Style

	// Glyphs
	SuccessGlyph string
	FailureGlyph string
	WarningGlyph string
	InfoGlyph    string
}

// StatusStyles for worktree status badges.
type StatusStyles struct {
	Active  lipgloss.Style
	Stale   lipgloss.Style
	Running lipgloss.Style
}

// HeaderStyles for the header component.
type HeaderStyles struct {
	Root      lipgloss.Style
	Logo      lipgloss.Style
	Title     lipgloss.Style
	Subtitle  lipgloss.Style
	Version   lipgloss.Style
	Container lipgloss.Style
}

// TabStyles for the tabs component.
type TabStyles struct {
	Tab               lipgloss.Style
	ActiveTab         lipgloss.Style
	TabSeparator      lipgloss.Style
	TabsRow           lipgloss.Style
	TabCount          lipgloss.Style
	OverflowIndicator lipgloss.Style
}

// SidebarStyles for the sidebar component.
type SidebarStyles struct {
	Root        lipgloss.Style
	Title       lipgloss.Style
	Content     lipgloss.Style
	PagerStyle  lipgloss.Style
	BorderWidth int
	PagerHeight int
	ContentPad  int
}

// TableStyles for the table component.
type TableStyles struct {
	CellStyle         lipgloss.Style
	SelectedCellStyle lipgloss.Style
	TitleCellStyle    lipgloss.Style
	HeaderStyle       lipgloss.Style
	RowStyle          lipgloss.Style
	EmptyState        lipgloss.Style
}

// SettingsStyles for the settings form.
type SettingsStyles struct {
	Category    lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	Selected    lipgloss.Style
	Description lipgloss.Style
	Cursor      lipgloss.Style
	Toggle      lipgloss.Style
	Input       lipgloss.Style
	InputFocus  lipgloss.Style
}

// HelpStyles for the help component.
type HelpStyles struct {
	Text         lipgloss.Style
	KeyText      lipgloss.Style
	BubbleStyles help.Styles
}

// InitStyles creates all styles based on the theme.
// Closely follows gh-dash's style initialization.
func InitStyles(t theme.Theme) Styles {
	var s Styles

	// Common styles
	s.Common.MainTextStyle = lipgloss.NewStyle().
		Foreground(t.PrimaryText).
		Bold(true)

	s.Common.FaintTextStyle = lipgloss.NewStyle().
		Foreground(t.FaintText)

	s.Common.AccentTextStyle = lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Bold(true)

	s.Common.ErrorStyle = lipgloss.NewStyle().
		Foreground(t.ErrorText).
		Bold(true)

	s.Common.SuccessStyle = lipgloss.NewStyle().
		Foreground(t.SuccessText).
		Bold(true)

	s.Common.WarningStyle = lipgloss.NewStyle().
		Foreground(t.WarningText)

	s.Common.SuccessGlyph = lipgloss.NewStyle().
		Foreground(t.SuccessText).
		Render(constants.SuccessIcon)

	s.Common.FailureGlyph = lipgloss.NewStyle().
		Foreground(t.ErrorText).
		Render(constants.FailureIcon)

	s.Common.WarningGlyph = lipgloss.NewStyle().
		Foreground(t.WarningText).
		Render(constants.WarningIcon)

	s.Common.InfoGlyph = lipgloss.NewStyle().
		Foreground(t.SecondaryText).
		Render(constants.InfoIcon)

	// Status styles - get colors based on theme
	statusColors := theme.GetStatusColors(t.Name)
	s.Status.Active = lipgloss.NewStyle().
		Foreground(statusColors.Active).
		Bold(true)
	s.Status.Stale = lipgloss.NewStyle().
		Foreground(statusColors.Stale)
	s.Status.Running = lipgloss.NewStyle().
		Foreground(statusColors.Running).
		Bold(true)

	// Header styles
	s.Header.Root = lipgloss.NewStyle().
		BorderStyle(lipgloss.ThickBorder()).
		BorderBottom(true).
		BorderForeground(t.PrimaryBorder)

	s.Header.Logo = lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Bold(true).
		Padding(0, 1)

	s.Header.Title = lipgloss.NewStyle().
		Foreground(t.PrimaryText).
		Bold(true).
		Padding(0, 1)

	s.Header.Subtitle = lipgloss.NewStyle().
		Foreground(t.FaintText).
		Italic(true)

	s.Header.Version = lipgloss.NewStyle().
		Foreground(t.SecondaryText).
		Padding(0, 1)

	s.Header.Container = lipgloss.NewStyle().
		Height(constants.TabsContentHeight)

	// Tab styles - gh-dash style
	s.Tabs.Tab = lipgloss.NewStyle().
		Faint(true).
		Padding(0, 2)

	s.Tabs.ActiveTab = lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Bold(true).
		Padding(0, 2)

	s.Tabs.TabSeparator = lipgloss.NewStyle().
		Foreground(t.SecondaryBorder)

	s.Tabs.TabsRow = lipgloss.NewStyle().
		Height(constants.TabsContentHeight)

	s.Tabs.TabCount = lipgloss.NewStyle().
		Foreground(t.FaintText)

	s.Tabs.OverflowIndicator = s.Common.FaintTextStyle.Bold(true).Padding(0, 1)

	// Sidebar styles - gh-dash style
	s.Sidebar.BorderWidth = 1
	s.Sidebar.PagerHeight = 1
	s.Sidebar.ContentPad = 2

	s.Sidebar.Root = lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.Border{
			Left: constants.BorderVertical,
		}).
		BorderForeground(t.PrimaryBorder)

	s.Sidebar.Title = lipgloss.NewStyle().
		Foreground(t.PrimaryText).
		Bold(true).
		Padding(0, 1)

	s.Sidebar.Content = lipgloss.NewStyle().
		Padding(1, 2)

	s.Sidebar.PagerStyle = lipgloss.NewStyle().
		Height(1).
		Bold(true).
		Foreground(t.FaintText)

	// Table styles - gh-dash style
	s.Table.CellStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1).
		MaxHeight(1)

	s.Table.SelectedCellStyle = s.Table.CellStyle.
		Background(t.SelectedBackground)

	s.Table.TitleCellStyle = s.Table.CellStyle.
		Bold(true).
		Foreground(t.PrimaryText)

	s.Table.HeaderStyle = lipgloss.NewStyle()

	s.Table.RowStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.FaintBorder)

	s.Table.EmptyState = lipgloss.NewStyle().
		Faint(true).
		PaddingLeft(1).
		MarginBottom(1)

	// Settings styles
	s.Settings.Category = lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Bold(true).
		MarginTop(1).
		MarginBottom(1)

	s.Settings.Label = lipgloss.NewStyle().
		Foreground(t.PrimaryText).
		Width(28)

	s.Settings.Value = lipgloss.NewStyle().
		Foreground(t.SecondaryText)

	s.Settings.Selected = lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Bold(true)

	s.Settings.Description = lipgloss.NewStyle().
		Foreground(t.FaintText).
		Italic(true).
		MarginLeft(30)

	s.Settings.Cursor = lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Bold(true)

	s.Settings.Toggle = lipgloss.NewStyle().
		Foreground(t.SuccessText)

	s.Settings.Input = lipgloss.NewStyle().
		Foreground(t.PrimaryText).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.SecondaryBorder)

	s.Settings.InputFocus = lipgloss.NewStyle().
		Foreground(t.PrimaryText).
		Padding(0, 1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.LogoColor)

	// Help styles - gh-dash style
	s.Help.Text = lipgloss.NewStyle().Foreground(t.SecondaryText)
	s.Help.KeyText = lipgloss.NewStyle().Foreground(t.PrimaryText)
	s.Help.BubbleStyles = help.Styles{
		ShortDesc:      s.Help.Text.Foreground(t.FaintText),
		FullDesc:       s.Help.Text.Foreground(t.FaintText),
		ShortSeparator: s.Help.Text.Foreground(t.SecondaryBorder),
		FullSeparator:  s.Help.Text,
		FullKey:        s.Help.KeyText,
		ShortKey:       s.Help.KeyText,
		Ellipsis:       s.Help.Text,
	}

	return s
}
