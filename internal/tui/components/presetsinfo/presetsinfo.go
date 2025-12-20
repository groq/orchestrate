package presetsinfo

import (
	"path/filepath"
	"strings"

	"orchestrate/config"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
	"orchestrate/internal/tui/theme"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	ctx        *context.ProgramContext
	dimensions constants.Dimensions
}

func New(ctx *context.ProgramContext) Model {
	return Model{ctx: ctx}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	if m.ctx == nil {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(m.ctx.Styles.Common.AccentTextStyle.Render("Manage Presets"))
	b.WriteString("\n\n")

	// Info section
	settingsPath := filepath.Join(m.ctx.DataDir, config.SettingsFileName)

	b.WriteString(m.ctx.Styles.Common.MainTextStyle.Render("Presets are managed via your settings.yaml file."))
	b.WriteString("\n\n")

	b.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render("File Location:"))
	b.WriteString("\n")
	pathStyle := lipgloss.NewStyle().
		Foreground(theme.LogoColor).
		Background(m.ctx.Theme.FaintBorder).
		Padding(0, 1).
		Bold(true)
	b.WriteString(pathStyle.Render(settingsPath))
	b.WriteString("\n\n")

	b.WriteString(m.ctx.Styles.Settings.Category.Render("How it works:"))
	b.WriteString("\n")
	instructions := []string{
		"- Each preset contains a list of agent worktrees to create.",
		"- You can specify an 'agent' name and an optional 'n' multiplier.",
		"- You can also add 'commands' to run in each worktree.",
		"- Changes to settings.yaml are applied automatically on the next launch.",
	}
	for _, inst := range instructions {
		b.WriteString(m.ctx.Styles.Common.MainTextStyle.Render(inst))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.ctx.Styles.Settings.Category.Render("Example configuration:"))
	b.WriteString("\n")

	exampleYAML := `default: dev
presets:
  dev:
    - agent: claude
      commands:
        - command: "./bin/go run ./cmd/myapp"
          title: "App"
    - agent: codex`

	exampleBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.ctx.Theme.FaintBorder).
		Padding(1).
		Width(50).
		Render(exampleYAML)

	b.WriteString(exampleBox)

	return lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.dimensions.Width).
		Height(m.dimensions.Height).
		Render(b.String())
}

func (m *Model) SetDimensions(dims constants.Dimensions) {
	m.dimensions = dims
}

func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
}
