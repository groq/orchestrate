package constants

import tea "github.com/charmbracelet/bubbletea"

// ErrMsg represents an error message.
type ErrMsg struct {
	Err error
}

// SettingsSavedMsg indicates settings were saved successfully.
type SettingsSavedMsg struct{}

// SettingsErrorMsg indicates an error saving settings.
type SettingsErrorMsg struct {
	Err error
}

// ViewChangedMsg indicates the view has changed.
type ViewChangedMsg struct {
	View ViewType
}

// RefreshMsg requests a refresh of the current view.
type RefreshMsg struct{}

// TickMsg is used for periodic updates.
type TickMsg struct{}

// ClearStatusMsg clears the status message.
type ClearStatusMsg struct{}

// StatusMsg sets a status message.
type StatusMsg struct {
	Message string
	IsError bool
}

// WindowSizeMsg wraps tea.WindowSizeMsg for internal use.
type WindowSizeMsg = tea.WindowSizeMsg

