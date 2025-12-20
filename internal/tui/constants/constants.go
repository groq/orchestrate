// Package constants provides shared constants for the TUI.
package constants

import (
	"github.com/charmbracelet/bubbles/key"
)

// Dimensions represents width and height measurements.
type Dimensions struct {
	Width  int
	Height int
}

// KeyMap defines the global key bindings.
type KeyMap struct {
	Up            key.Binding
	Down          key.Binding
	Left          key.Binding
	Right         key.Binding
	FirstItem     key.Binding
	LastItem      key.Binding
	Enter         key.Binding
	Back          key.Binding
	Tab           key.Binding
	TogglePreview key.Binding
	Help          key.Binding
	Quit          key.Binding
	Save          key.Binding
	Search        key.Binding
	Refresh       key.Binding
	Delete        key.Binding
}

// Layout constants
const (
	HeaderHeight       = 1
	TabsHeight         = 1
	TabsContentHeight  = 1
	SidebarWidth       = 50
	MainContentPadding = 1
	ExpandedHelpHeight = 12
	InputBoxHeight     = 5
	TableHeaderHeight  = 2
	SingleRuneWidth    = 4
)

// Icons and glyphs - Nerd Fonts compatible
const (
	Ellipsis = "..."

	// Status icons
	SuccessIcon = "[+]"
	FailureIcon = "[x]"
	WarningIcon = "[!]"
	InfoIcon    = "[i]"
	WaitingIcon = "[...]"
	EmptyIcon   = "[ ]"

	// Navigation icons
	ArrowRight = ">"
	ArrowLeft  = "<"
	ArrowUp    = "^"
	ArrowDown  = "v"
	Cursor     = ">"
	CursorFull = ">>"

	// App icons
	AgentIcon    = ""
	PresetIcon   = ""
	SettingIcon  = ""
	TermIcon     = ""
	FolderIcon   = ""
	BranchIcon   = ""
	ClockIcon    = ""
	SaveIcon     = "[s]"
	WorktreeIcon = ""
	RefreshIcon  = "[r]"
	DeleteIcon   = "[d]"

	// Terminal type icons
	ITermIcon    = ""
	TerminalIcon = ""

	// Toggle icons
	ToggleOn  = "[X]"
	ToggleOff = "[ ]"
	CheckOn   = "[X]"
	CheckOff  = "[ ]"

	// Border chars
	BorderVertical   = "|"
	BorderHorizontal = "-"
	BorderTopLeft    = "+"
	BorderTopRight   = "+"
	BorderBotLeft    = "+"
	BorderBotRight   = "+"

	// Logo - Orchestrate branding
	Logo = `ORCHESTRATE`

	LogoSmall = "orchestrate"
)

// View types
type ViewType int

const (
	WorktreesView ViewType = iota
	LaunchView
	SettingsView
	PresetsView
)

// String returns the string representation of the view type.
func (v ViewType) String() string {
	switch v {
	case WorktreesView:
		return "Worktrees"
	case LaunchView:
		return "Launch"
	case PresetsView:
		return "Presets"
	case SettingsView:
		return "Settings"
	default:
		return "Unknown"
	}
}

// Icon returns the icon for the view type.
func (v ViewType) Icon() string {
	switch v {
	case WorktreesView:
		return WorktreeIcon
	case LaunchView:
		return AgentIcon
	case PresetsView:
		return PresetIcon
	case SettingsView:
		return SettingIcon
	default:
		return ""
	}
}
