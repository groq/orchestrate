// Package keys provides key bindings for the TUI.
package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all key bindings for the application.
type KeyMap struct {
	// Navigation
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	FirstItem key.Binding
	LastItem  key.Binding

	// Actions
	Enter   key.Binding
	Back    key.Binding
	Tab     key.Binding
	Save    key.Binding
	Refresh key.Binding
	Delete  key.Binding
	Edit    key.Binding

	// Toggles
	TogglePreview key.Binding
	Help          key.Binding

	// App
	Quit key.Binding
}

// Keys is the global key map.
var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "right"),
	),
	FirstItem: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "first"),
	),
	LastItem: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "last"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next view"),
	),
	Save: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "refresh"),
	),
	Delete: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "delete"),
	),
	Edit: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "edit"),
	),
	TogglePreview: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "preview"),
	),
	Help: key.NewBinding(
		key.WithKeys("ctrl+?"),
		key.WithHelp("ctrl+?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}

// ShortHelp returns a minimal set of key bindings for the short help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Refresh, k.Help, k.Quit}
}

// FullHelp returns the complete set of key bindings for the expanded help.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Tab},
		{k.Enter, k.Back, k.Refresh},
		{k.TogglePreview, k.Help, k.Quit},
	}
}

// NavigationKeys returns only navigation keys.
func NavigationKeys() []key.Binding {
	return []key.Binding{Keys.Up, Keys.Down, Keys.Left, Keys.Right}
}

// ActionKeys returns action keys.
func ActionKeys() []key.Binding {
	return []key.Binding{Keys.Enter, Keys.Back, Keys.Save, Keys.Refresh}
}
