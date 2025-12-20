// Package launch provides the launch session component.
package launch

import (
	"fmt"
	"strings"

	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
	"orchestrate/internal/tui/theme"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Field represents an input field.
type Field int

const (
	FieldRepo Field = iota
	FieldName
	FieldPrompt
	FieldPreset
	FieldLaunch
	FieldCount
)

// Layout constants for consistent field alignment
const (
	CursorWidth     = 2                                                                  // "▸ " or "  "
	LabelWidth      = 12                                                                 // Fixed width for field labels
	InputWidth      = 44                                                                 // Internal content width for inputs
	BorderWidth     = 2                                                                  // Border takes 2 chars (left + right)
	PaddingWidth    = 2                                                                  // Padding is 1 char each side (total 2)
	TotalFieldWidth = CursorWidth + LabelWidth + InputWidth + BorderWidth + PaddingWidth // = 62
)

// LaunchRequestMsg is sent when the user wants to launch a session.
type LaunchRequestMsg struct {
	Repo   string
	Name   string
	Prompt string
	Preset string
}

// Model represents the launch component.
type Model struct {
	ctx        *context.ProgramContext
	repoInput  textinput.Model
	nameInput  textinput.Model
	promptArea textarea.Model
	focused    Field
	presets    []string
	presetIdx  int
	dimensions constants.Dimensions
}

// New creates a new launch model.
func New(ctx *context.ProgramContext) Model {
	// Repo input
	repoInput := textinput.New()
	repoInput.Placeholder = "owner/repo"
	repoInput.CharLimit = 100
	repoInput.Width = InputWidth - 2 // align with box interior (padding 1 each side)
	repoInput.Prompt = ""
	repoInput.Focus()

	// Name input
	nameInput := textinput.New()
	nameInput.Placeholder = "feature-name"
	nameInput.CharLimit = 50
	nameInput.Width = InputWidth - 2 // align with box interior (padding 1 each side)
	nameInput.Prompt = ""

	// Prompt textarea
	promptArea := textarea.New()
	promptArea.Placeholder = "What should the agent do?"
	promptArea.CharLimit = 2000
	promptArea.SetWidth(InputWidth - 2) // align with box interior (padding 1 each side)
	promptArea.SetHeight(3)
	promptArea.ShowLineNumbers = false
	promptArea.Prompt = "" // remove default gutter
	promptArea.FocusedStyle.CursorLine = lipgloss.NewStyle()
	promptArea.BlurredStyle.CursorLine = lipgloss.NewStyle()
	promptArea.FocusedStyle.Base = lipgloss.NewStyle()
	promptArea.BlurredStyle.Base = lipgloss.NewStyle()
	promptArea.FocusedStyle.Text = lipgloss.NewStyle()
	promptArea.BlurredStyle.Text = lipgloss.NewStyle()

	// Get presets from context
	var presets []string
	if ctx != nil {
		presets = ctx.GetPresetNames()
	}
	if len(presets) == 0 {
		presets = []string{"default"}
	}

	return Model{
		ctx:        ctx,
		repoInput:  repoInput,
		nameInput:  nameInput,
		promptArea: promptArea,
		focused:    FieldRepo,
		presets:    presets,
		presetIdx:  0,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+enter":
			return m, m.submit()

		case "tab":
			m.nextField()
			return m, nil

		case "shift+tab":
			m.prevField()
			return m, nil

		case "up":
			// Allow up to navigate between fields (not in prompt)
			if m.focused != FieldPrompt {
				m.prevField()
				return m, nil
			}
			// In prompt, check if at first line - if so, go to prev field
			lines := strings.Split(m.promptArea.Value(), "\n")
			if len(lines) <= 1 || m.promptArea.Line() == 0 {
				m.prevField()
				return m, nil
			}

		case "down":
			// Allow down to navigate between fields (not in prompt)
			if m.focused != FieldPrompt {
				m.nextField()
				return m, nil
			}
			// In prompt, check if at last line - if so, go to next field
			lines := strings.Split(m.promptArea.Value(), "\n")
			if len(lines) <= 1 || m.promptArea.Line() >= len(lines)-1 {
				m.nextField()
				return m, nil
			}

		case "enter":
			if m.focused == FieldLaunch {
				return m, m.submit()
			}
			// For other fields, move to next (except prompt which handles enter itself)
			if m.focused != FieldPrompt {
				m.nextField()
				return m, nil
			}

		case "left":
			if m.focused == FieldPreset {
				m.prevPreset()
				return m, nil
			}

		case "right":
			if m.focused == FieldPreset {
				m.nextPreset()
				return m, nil
			}
		}
	}

	// Update focused input
	var cmd tea.Cmd
	switch m.focused {
	case FieldRepo:
		m.repoInput, cmd = m.repoInput.Update(msg)
		cmds = append(cmds, cmd)
	case FieldName:
		m.nameInput, cmd = m.nameInput.Update(msg)
		cmds = append(cmds, cmd)
	case FieldPrompt:
		m.promptArea, cmd = m.promptArea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) nextField() {
	m.blurCurrent()
	m.focused++
	if m.focused >= FieldCount {
		m.focused = 0
	}
	m.focusCurrent()
}

func (m *Model) prevField() {
	m.blurCurrent()
	if m.focused == 0 {
		m.focused = FieldCount - 1
	} else {
		m.focused--
	}
	m.focusCurrent()
}

func (m *Model) blurCurrent() {
	switch m.focused {
	case FieldRepo:
		m.repoInput.Blur()
	case FieldName:
		m.nameInput.Blur()
	case FieldPrompt:
		m.promptArea.Blur()
	}
}

func (m *Model) focusCurrent() {
	switch m.focused {
	case FieldRepo:
		m.repoInput.Focus()
	case FieldName:
		m.nameInput.Focus()
	case FieldPrompt:
		m.promptArea.Focus()
	}
}

func (m *Model) nextPreset() {
	m.presetIdx++
	if m.presetIdx >= len(m.presets) {
		m.presetIdx = 0
	}
}

func (m *Model) prevPreset() {
	m.presetIdx--
	if m.presetIdx < 0 {
		m.presetIdx = len(m.presets) - 1
	}
}

func (m Model) submit() tea.Cmd {
	repo := strings.TrimSpace(m.repoInput.Value())
	name := strings.TrimSpace(m.nameInput.Value())
	prompt := strings.TrimSpace(m.promptArea.Value())
	preset := m.presets[m.presetIdx]

	if repo == "" || name == "" || prompt == "" {
		return nil
	}

	return func() tea.Msg {
		return LaunchRequestMsg{
			Repo:   repo,
			Name:   name,
			Prompt: prompt,
			Preset: preset,
		}
	}
}

// View renders the launch view.
func (m Model) View() string {
	if m.ctx == nil {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(m.ctx.Styles.Common.AccentTextStyle.Render("Launch New Session"))
	b.WriteString("\n")
	b.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render("Create worktrees and start AI coding agents"))
	b.WriteString("\n\n")

	// Form fields with consistent layout
	b.WriteString(m.renderField("Repository", "e.g., groq/orion", m.repoInput.View(), FieldRepo))
	b.WriteString(m.renderField("Branch", "e.g., fix-bug", m.nameInput.View(), FieldName))
	b.WriteString(m.renderPromptField())
	b.WriteString(m.renderPresetField())

	// Submit button
	b.WriteString("\n")
	isActive := m.focused == FieldLaunch

	btnStyle := lipgloss.NewStyle().
		Background(theme.LogoColor).
		Foreground(lipgloss.Color("#000")).
		Bold(true).
		Padding(0, 2)

	if isActive {
		btnStyle = btnStyle.
			Background(lipgloss.Color("#FFF")).
			Foreground(theme.LogoColor).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.LogoColor).
			Padding(0, 1)

		cursor := lipgloss.NewStyle().Foreground(theme.LogoColor).Render("> ")
		b.WriteString(cursor + btnStyle.Render("Launch"))
	} else {
		b.WriteString("  ")
		b.WriteString(btnStyle.Render("Launch"))
	}
	b.WriteString("\n\n")

	// Help
	b.WriteString("  ")
	b.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render("Arrows/Tab"))
	b.WriteString(" ")
	b.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render("navigate"))
	b.WriteString("  ")
	b.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render("Enter"))
	b.WriteString(" ")
	b.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render("select/launch"))

	return lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.dimensions.Width).
		Height(m.dimensions.Height).
		Render(b.String())
}

// renderFieldBase provides a unified field rendering system with consistent alignment.
// contentView should be the pre-rendered content (textinput, textarea, or custom selector).
// Labels and hints are rendered above the input box to keep boxes aligned.
func (m Model) renderFieldBase(label, hint, contentView string, field Field) string {
	isActive := m.focused == field

	// Cursor: "▸ " if active, "  " otherwise
	cursor := "  "
	if isActive {
		cursor = lipgloss.NewStyle().Foreground(theme.LogoColor).Render("▸ ")
	}

	// Label with fixed width
	labelStyle := lipgloss.NewStyle().Width(LabelWidth)
	if isActive {
		labelStyle = labelStyle.Foreground(theme.LogoColor).Bold(true)
	}
	renderedLabel := labelStyle.Render(label)

	// Input box styling
	boxStyle := lipgloss.NewStyle().
		Width(InputWidth).
		Padding(0, 1)
	if isActive {
		boxStyle = boxStyle.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.LogoColor)
	} else {
		boxStyle = boxStyle.
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.ctx.Theme.FaintBorder)
	}
	renderedBox := boxStyle.Render(contentView)

	// Hint styling
	hintStyle := m.ctx.Styles.Common.FaintTextStyle.MarginLeft(1)
	renderedHint := ""
	if hint != "" {
		renderedHint = " " + hintStyle.Render(hint)
	}

	// Label and hint on first line, box on second line (box border at col 0)
	line1 := cursor + renderedLabel + renderedHint
	line2 := renderedBox
	return line1 + "\n" + line2 + "\n\n"
}

func (m Model) renderField(label, hint, input string, field Field) string {
	return m.renderFieldBase(label, hint, input, field)
}

func (m Model) renderPromptField() string {
	return m.renderFieldBase("Prompt", "Instructions for the AI", m.promptArea.View(), FieldPrompt)
}

func (m Model) renderPresetField() string {
	isActive := m.focused == FieldPreset

	arrowStyle := m.ctx.Styles.Common.FaintTextStyle
	if isActive {
		arrowStyle = lipgloss.NewStyle().Foreground(theme.LogoColor).Bold(true)
	}
	presetName := m.presets[m.presetIdx]
	countStr := fmt.Sprintf("(%d/%d)", m.presetIdx+1, len(m.presets))

	// Calculate available width for preset name (centered)
	// Total: 44, Left arrow: 2, Right arrow: 2, Count with padding: len(countStr) + 1
	availableWidth := InputWidth - 4 - len(countStr) - 1

	presetStyle := lipgloss.NewStyle().Width(availableWidth).Align(lipgloss.Center)
	if isActive {
		presetStyle = presetStyle.Foreground(theme.LogoColor).Bold(true)
	}

	// Build selector without the box (renderFieldBase will add it)
	selector := arrowStyle.Render("<") + " " +
		presetStyle.Render(presetName) + " " +
		arrowStyle.Render(">") + " " +
		m.ctx.Styles.Common.FaintTextStyle.Render(countStr)

	// Use renderFieldBase but without the hint since selector already has the count
	hint := "Arrows to cycle presets"
	return m.renderFieldBase("Preset", hint, selector, FieldPreset)
}

// SetDimensions sets the component dimensions.
func (m *Model) SetDimensions(dims constants.Dimensions) {
	m.dimensions = dims
}

// UpdateProgramContext updates the context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
	if ctx != nil {
		m.presets = ctx.GetPresetNames()
		if len(m.presets) == 0 {
			m.presets = []string{"default"}
		}
	}
}

// Reset clears the form.
func (m *Model) Reset() {
	m.repoInput.Reset()
	m.nameInput.Reset()
	m.promptArea.Reset()
	m.focused = FieldRepo
	m.repoInput.Focus()
}

// GetValues returns the current form values.
func (m Model) GetValues() (repo, name, prompt, preset string) {
	return m.repoInput.Value(),
		m.nameInput.Value(),
		m.promptArea.Value(),
		m.presets[m.presetIdx]
}

// IsAtTop returns true if the focus is at the first field.
func (m Model) IsAtTop() bool {
	return m.focused == FieldRepo
}

// ConsumesKey returns true if the component wants to handle the key message exclusively.
func (m Model) ConsumesKey(msg tea.KeyMsg) bool {
	if msg.String() == "left" || msg.String() == "right" {
		return m.focused == FieldPreset
	}
	return false
}
