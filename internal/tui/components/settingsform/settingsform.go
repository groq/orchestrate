// Package settingsform provides the settings form component.
package settingsform

import (
	"fmt"
	"strconv"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
	"orchestrate/internal/tui/keys"
	"orchestrate/internal/tui/theme"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SettingType represents the type of a setting.
type SettingType int

const (
	TypeSelect SettingType = iota
	TypeToggle
	TypeNumber
	TypeText
)

// Setting represents a single setting.
type Setting struct {
	Key         string
	Label       string
	Description string
	Type        SettingType
	Value       interface{}
	Options     []string
}

// Category represents a group of settings.
type Category struct {
	Name     string
	Icon     string
	Settings []Setting
}

// Model represents the settings form component.
type Model struct {
	ctx           *context.ProgramContext
	categories    []Category
	categoryIndex int
	settingIndex  int
	editing       bool
	textInput     textinput.Model
	dimensions    constants.Dimensions
}

// New creates a new settings form model.
func New(ctx *context.ProgramContext) Model {
	ti := textinput.New()
	ti.CharLimit = 64
	ti.Width = 30

	m := Model{
		ctx:       ctx,
		textInput: ti,
	}

	m.buildCategories()
	return m
}

func (m *Model) buildCategories() {
	if m.ctx == nil || m.ctx.AppSettings == nil {
		return
	}

	settings := m.ctx.AppSettings

	m.categories = []Category{
		{
			Name: "Terminal",
			Icon: constants.TermIcon,
			Settings: []Setting{
				{
					Key:         "terminal.type",
					Label:       "Terminal Type",
					Description: "Choose between iTerm2 and regular terminal",
					Type:        TypeSelect,
					Value:       string(settings.Terminal.Type),
					Options:     []string{"iterm2", "terminal"},
				},
				{
					Key:         "terminal.maximize",
					Label:       "Maximize on Launch",
					Description: "Maximize terminal windows when launching sessions",
					Type:        TypeToggle,
					Value:       settings.Terminal.MaximizeOnLaunch,
				},
			},
		},
		{
			Name: "User Interface",
			Icon: constants.SettingIcon,
			Settings: []Setting{
				{
					Key:         "ui.theme",
					Label:       "Theme",
					Description: "Color theme for the TUI",
					Type:        TypeSelect,
					Value:       settings.UI.Theme,
					Options:     config.GetThemeOptions(),
				},
			},
		},
		{
			Name: "Session",
			Icon: constants.AgentIcon,
			Settings: []Setting{
				{
					Key:         "session.default_preset",
					Label:       "Default Preset",
					Description: "Preset to use when none is specified",
					Type:        TypeText,
					Value:       settings.Session.DefaultPreset,
				},
				{
					Key:         "session.auto_clean",
					Label:       "Auto Clean Worktrees",
					Description: "Automatically remove old worktrees",
					Type:        TypeToggle,
					Value:       settings.Session.AutoCleanWorktrees,
				},
				{
					Key:         "session.retention_days",
					Label:       "Worktree Retention (days)",
					Description: "How many days to keep old worktrees (1-365)",
					Type:        TypeNumber,
					Value:       settings.Session.WorktreeRetentionDays,
				},
			},
		},
	}
}

// Init initializes the settings form.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			return m.handleEditing(msg)
		}
		return m.handleNavigation(msg)
	}
	return m, nil
}

func (m Model) handleNavigation(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, keys.Keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, keys.Keys.Enter):
		return m.startEditing()
	case msg.String() == "left":
		m.adjustValue(-1)
	case msg.String() == "right":
		m.adjustValue(1)
	case key.Matches(msg, keys.Keys.Tab):
		m.nextCategory()
	}
	return m, nil
}

func (m Model) handleEditing(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.finishEditing()
		return m, nil
	case "esc":
		m.editing = false
		return m, nil
	}

	setting := m.getCurrentSetting()
	if setting == nil {
		m.editing = false
		return m, nil
	}

	switch setting.Type {
	case TypeSelect, TypeNumber, TypeToggle:
		if msg.String() == "left" {
			m.adjustValue(-1)
		} else if msg.String() == "right" {
			m.adjustValue(1)
		}
	case TypeText:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) moveCursor(delta int) {
	if len(m.categories) == 0 {
		return
	}

	category := m.categories[m.categoryIndex]
	newIndex := m.settingIndex + delta

	if newIndex < 0 {
		if m.categoryIndex > 0 {
			m.categoryIndex--
			m.settingIndex = len(m.categories[m.categoryIndex].Settings) - 1
		}
	} else if newIndex >= len(category.Settings) {
		if m.categoryIndex < len(m.categories)-1 {
			m.categoryIndex++
			m.settingIndex = 0
		}
	} else {
		m.settingIndex = newIndex
	}
}

func (m *Model) nextCategory() {
	if len(m.categories) == 0 {
		return
	}
	m.categoryIndex = (m.categoryIndex + 1) % len(m.categories)
	m.settingIndex = 0
}

func (m *Model) adjustValue(delta int) {
	setting := m.getCurrentSetting()
	if setting == nil {
		return
	}

	switch setting.Type {
	case TypeSelect:
		current := setting.Value.(string)
		idx := 0
		for i, opt := range setting.Options {
			if opt == current {
				idx = i
				break
			}
		}
		idx = (idx + delta + len(setting.Options)) % len(setting.Options)
		m.updateSettingValue(setting.Key, setting.Options[idx])

	case TypeToggle:
		current := setting.Value.(bool)
		m.updateSettingValue(setting.Key, !current)

	case TypeNumber:
		current := setting.Value.(int)
		newVal := current + delta
		if newVal < 1 {
			newVal = 1
		}
		if newVal > 365 {
			newVal = 365
		}
		m.updateSettingValue(setting.Key, newVal)
	}

	m.buildCategories()
	m.autoSave()
}

func (m *Model) autoSave() {
	if m.ctx != nil && m.ctx.AppSettings != nil {
		_ = config.SaveAppSettings(m.ctx.DataDir, m.ctx.AppSettings)
	}
}

func (m Model) startEditing() (Model, tea.Cmd) {
	setting := m.getCurrentSetting()
	if setting == nil {
		return m, nil
	}

	switch setting.Type {
	case TypeText, TypeNumber:
		m.editing = true
		m.textInput.SetValue(fmt.Sprintf("%v", setting.Value))
		m.textInput.Focus()
		return m, textinput.Blink
	case TypeSelect, TypeToggle:
		m.adjustValue(1)
	}
	return m, nil
}

func (m *Model) finishEditing() {
	setting := m.getCurrentSetting()
	if setting == nil {
		m.editing = false
		return
	}

	value := m.textInput.Value()

	switch setting.Type {
	case TypeText:
		m.updateSettingValue(setting.Key, value)
	case TypeNumber:
		if num, err := strconv.Atoi(value); err == nil {
			if num < 1 {
				num = 1
			}
			if num > 365 {
				num = 365
			}
			m.updateSettingValue(setting.Key, num)
		}
	}

	m.editing = false
	m.buildCategories()
	m.autoSave()
}

func (m *Model) getCurrentSetting() *Setting {
	if m.categoryIndex >= len(m.categories) {
		return nil
	}
	category := m.categories[m.categoryIndex]
	if m.settingIndex >= len(category.Settings) {
		return nil
	}
	return &category.Settings[m.settingIndex]
}

func (m *Model) updateSettingValue(key string, value interface{}) {
	if m.ctx == nil || m.ctx.AppSettings == nil {
		return
	}

	settings := m.ctx.AppSettings

	switch key {
	case "terminal.type":
		settings.Terminal.Type = config.TerminalType(value.(string))
	case "terminal.maximize":
		settings.Terminal.MaximizeOnLaunch = value.(bool)
	case "ui.status_bar":
		settings.UI.ShowStatusBar = value.(bool)
	case "ui.theme":
		settings.UI.Theme = value.(string)
		m.ctx.UpdateTheme(settings.UI.Theme)
	case "session.default_preset":
		settings.Session.DefaultPreset = value.(string)
	case "session.auto_clean":
		settings.Session.AutoCleanWorktrees = value.(bool)
	case "session.retention_days":
		settings.Session.WorktreeRetentionDays = value.(int)
	}
}

// View renders the settings form.
func (m Model) View() string {
	if m.ctx == nil {
		return ""
	}

	var sections []string

	// Title
	title := m.ctx.Styles.Common.AccentTextStyle.Render("Settings")
	subtitle := m.ctx.Styles.Common.FaintTextStyle.Render("Use Left/Right to change, Enter to edit")
	sections = append(sections, title, subtitle, "")

	globalIdx := 0
	currentGlobalIdx := m.getGlobalIndex()

	for catIdx, category := range m.categories {
		// Category header
		isCurrentCat := catIdx == m.categoryIndex
		catStyle := m.ctx.Styles.Settings.Category
		if isCurrentCat {
			catStyle = catStyle.Foreground(theme.LogoColor)
		}
		sections = append(sections, catStyle.Render(fmt.Sprintf("--- %s ---", category.Name)))

		for _, setting := range category.Settings {
			isSelected := globalIdx == currentGlobalIdx
			sections = append(sections, m.renderSetting(setting, isSelected))
			globalIdx++
		}
		sections = append(sections, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.dimensions.Width).
		Height(m.dimensions.Height).
		Render(content)
}

func (m Model) getGlobalIndex() int {
	idx := 0
	for i := 0; i < m.categoryIndex; i++ {
		idx += len(m.categories[i].Settings)
	}
	return idx + m.settingIndex
}

func (m Model) renderSetting(setting Setting, isSelected bool) string {
	// Cursor
	cursor := "  "
	if isSelected {
		cursor = m.ctx.Styles.Settings.Cursor.Render(constants.Cursor + " ")
	}

	// Label
	labelStyle := m.ctx.Styles.Settings.Label
	if isSelected {
		labelStyle = labelStyle.Foreground(theme.LogoColor).Bold(true)
	}
	label := labelStyle.Render(setting.Label + ":")

	// Value
	var value string
	if m.editing && isSelected && (setting.Type == TypeText || setting.Type == TypeNumber) {
		value = m.textInput.View()
	} else {
		value = m.formatValue(setting)
		valueStyle := m.ctx.Styles.Settings.Value
		if isSelected {
			valueStyle = m.ctx.Styles.Settings.Selected
		}
		value = valueStyle.Render(value)
	}

	// Hint
	hint := ""
	if isSelected {
		hintStyle := m.ctx.Styles.Common.FaintTextStyle
		if m.editing {
			hint = hintStyle.Render("  (Enter/Esc to finish)")
		} else {
			hint = hintStyle.Render("  (Arrows to change, Enter to edit)")
		}
	}

	line := cursor + label + " " + value + hint

	// Description
	desc := m.ctx.Styles.Settings.Description.Render(setting.Description)

	return line + "\n" + desc
}

func (m Model) formatValue(setting Setting) string {
	switch setting.Type {
	case TypeToggle:
		if setting.Value.(bool) {
			return constants.ToggleOn + " Enabled"
		}
		return constants.ToggleOff + " Disabled"
	case TypeSelect:
		return setting.Value.(string)
	case TypeNumber:
		return fmt.Sprintf("%d", setting.Value.(int))
	default:
		return fmt.Sprintf("%v", setting.Value)
	}
}

// SetDimensions sets the form dimensions.
func (m *Model) SetDimensions(dimensions constants.Dimensions) {
	m.dimensions = dimensions
}

// UpdateProgramContext updates the context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
	m.buildCategories()
}

// GetSettings returns the current settings.
func (m Model) GetSettings() *config.AppSettings {
	if m.ctx != nil {
		return m.ctx.AppSettings
	}
	return nil
}

// IsEditing returns true if currently editing a field.
func (m Model) IsEditing() bool {
	return m.editing
}

// IsAtTop returns true if the focus is at the first field.
func (m Model) IsAtTop() bool {
	return m.categoryIndex == 0 && m.settingIndex == 0 && !m.editing
}

// ConsumesKey returns true if the component wants to handle the key message exclusively.
func (m Model) ConsumesKey(msg tea.KeyMsg) bool {
	if m.editing {
		return msg.String() == "left" || msg.String() == "right" || msg.String() == "esc" || msg.String() == "enter"
	}
	return false
}
