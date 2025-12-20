// Package tui provides the terminal user interface for orchestrate.
// Styled after gh-dash for a polished, professional look.
package tui

import (
	"fmt"
	"os"
	"strings"

	"orchestrate/config"
	"orchestrate/internal/tui/components/header"
	"orchestrate/internal/tui/components/launch"
	"orchestrate/internal/tui/components/presetsinfo"
	"orchestrate/internal/tui/components/settingsform"
	"orchestrate/internal/tui/components/sidebar"
	"orchestrate/internal/tui/components/worktrees"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
	"orchestrate/internal/tui/keys"
	"orchestrate/launcher"
	"orchestrate/terminal"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type FocusArea int

const (
	FocusContent FocusArea = iota
	FocusHeader
)

// Model is the main TUI model.
type Model struct {
	ctx *context.ProgramContext

	// Components
	header       header.Model
	sidebar      sidebar.Model
	settingsForm settingsform.Model
	presetsInfo  presetsinfo.Model
	worktreeList worktrees.Model
	launchForm   launch.Model

	// State
	ready     bool
	launching bool
	focus     FocusArea
}

// NewModel creates a new TUI model.
func NewModel(dataDir string, appSettings *config.AppSettings, presetConfig *config.Config) Model {
	ctx := context.NewProgramContext(dataDir, appSettings, presetConfig)

	return Model{
		ctx:          ctx,
		header:       header.New(ctx),
		sidebar:      sidebar.New(ctx),
		settingsForm: settingsform.New(ctx),
		presetsInfo:  presetsinfo.New(ctx),
		worktreeList: worktrees.New(ctx),
		launchForm:   launch.New(ctx),
		ready:        false,
		launching:    false,
		focus:        FocusContent,
	}
}

// NewUIModel is an alias for NewModel for compatibility.
func NewUIModel(dataDir string, appSettings *config.AppSettings, presetConfig *config.Config) Model {
	return NewModel(dataDir, appSettings, presetConfig)
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.worktreeList.Refresh(),
	)
}

// Update handles all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ctx.UpdateWindowSize(msg)
		m.ready = true
		m.syncComponentDimensions()
		m.syncProgramContext()

	case tea.KeyMsg:
		// Global keys
		switch {
		case key.Matches(msg, keys.Keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Keys.Help):
			m.ctx.ToggleHelp()
			m.syncComponentDimensions()
			return m, nil

		case key.Matches(msg, keys.Keys.TogglePreview):
			m.sidebar.Toggle()
			m.ctx.ToggleSidebar()
			m.syncComponentDimensions()
			return m, nil

		case key.Matches(msg, keys.Keys.Save):
			if m.ctx.View == constants.SettingsView {
				return m, m.saveSettings()
			}
		}

		// View-specific handling
		var cmd tea.Cmd
		consumed := false

		if m.focus == FocusHeader {
			switch msg.String() {
			case "down":
				m.focus = FocusContent
				m.header.SetFocused(false)
				return m, nil
			case "left":
				m.prevView()
				m.focus = FocusHeader
				m.header.SetFocused(true)
				return m, nil
			case "right":
				m.nextView()
				m.focus = FocusHeader
				m.header.SetFocused(true)
				return m, nil
			}
		} else {
			// FocusContent
			switch m.ctx.View {
			case constants.WorktreesView:
				if msg.String() == "up" && m.worktreeList.IsAtTop() {
					m.focus = FocusHeader
					m.header.SetFocused(true)
					return m, nil
				}
				m.worktreeList, cmd = m.worktreeList.Update(msg)
			case constants.LaunchView:
				if msg.String() == "up" && m.launchForm.IsAtTop() {
					m.focus = FocusHeader
					m.header.SetFocused(true)
					return m, nil
				}
				consumed = m.launchForm.ConsumesKey(msg)
				m.launchForm, cmd = m.launchForm.Update(msg)
			case constants.SettingsView:
				if msg.String() == "up" && m.settingsForm.IsAtTop() {
					m.focus = FocusHeader
					m.header.SetFocused(true)
					return m, nil
				}
				consumed = m.settingsForm.ConsumesKey(msg)
				m.settingsForm, cmd = m.settingsForm.Update(msg)
			case constants.PresetsView:
				// PresetsView is just info, always go to header on up
				if msg.String() == "up" {
					m.focus = FocusHeader
					m.header.SetFocused(true)
					return m, nil
				}
				m.presetsInfo, cmd = m.presetsInfo.Update(msg)
			}
		}

		// If the view didn't consume the key (or if it's a global nav key), handle it here
		if !consumed && cmd == nil {
			switch {
			case key.Matches(msg, keys.Keys.Tab):
				m.nextView()
				return m, nil

			case key.Matches(msg, keys.Keys.Back):
				if m.ctx.View != constants.WorktreesView {
					m.ctx.SetView(constants.WorktreesView)
					m.syncProgramContext()
					return m, nil
				}
			}
		}

		return m, cmd

	case worktrees.WorktreesLoadedMsg:
		var cmd tea.Cmd
		m.worktreeList, cmd = m.worktreeList.Update(msg)
		return m, cmd

	case worktrees.FocusWorktreeMsg:
		// Focus existing iTerm window for this worktree
		if msg.Worktree != nil {
			m.ctx.SetStatus(fmt.Sprintf("Focusing window for: %s...", msg.Worktree.Name), false)
			return m, m.doFocusWorktree(msg.Worktree)
		}
		return m, nil

	case worktrees.WorktreeDetailsMsg:
		// Show detailed worktree info in sidebar
		if msg.Worktree != nil {
			details := m.renderWorktreeDetails(msg.Worktree)
			m.sidebar.SetContent(details)
			if !m.sidebar.IsOpen {
				m.sidebar.Toggle()
				m.ctx.ToggleSidebar()
				m.syncComponentDimensions()
			}
		}
		return m, nil

	case worktrees.CloseWorktreeDetailsMsg:
		// Close the sidebar
		if m.sidebar.IsOpen {
			m.sidebar.Toggle()
			m.ctx.ToggleSidebar()
			m.syncComponentDimensions()
		}
		return m, nil

	case worktrees.OpenWorktreeMsg:
		// Open worktree in iTerm with same preset
		if msg.Worktree != nil && msg.Worktree.HasMeta {
			m.ctx.SetStatus(fmt.Sprintf("Opening worktree: %s...", msg.Worktree.Name), false)
			return m, m.doReopenWorktree(msg.Worktree)
		} else if msg.Worktree != nil {
			m.ctx.SetStatus("No preset info available for this worktree", true)
		}
		return m, nil

	case worktrees.DeleteWorktreeMsg:
		// Delete worktree
		if msg.Worktree != nil {
			m.ctx.SetStatus(fmt.Sprintf("Deleting worktree: %s...", msg.Worktree.Name), false)
			return m, m.doDeleteWorktree(msg.Worktree)
		}
		return m, nil

	case worktrees.WorktreeDeletedMsg:
		var cmd tea.Cmd
		m.worktreeList, cmd = m.worktreeList.Update(msg)
		if msg.Err != nil {
			m.ctx.SetStatus(fmt.Sprintf("Delete failed: %v", msg.Err), true)
		} else {
			m.ctx.SetStatus("Worktree deleted successfully", false)
		}
		return m, cmd

	case launch.LaunchRequestMsg:
		// Handle launch request
		m.launching = true
		m.ctx.SetStatus("Launching session...", false)
		return m, m.doLaunch(msg)

	case LaunchResultMsg:
		m.launching = false
		if msg.Err != nil {
			m.ctx.SetStatus(fmt.Sprintf("Launch failed: %v", msg.Err), true)
		} else {
			m.ctx.SetStatus(fmt.Sprintf("Launched %d session(s) in %d worktree(s)!", msg.Sessions, msg.Worktrees), false)
			m.launchForm.Reset()
			m.focus = FocusContent
			m.header.SetFocused(false)
			// Refresh worktrees to show new ones
			return m, m.worktreeList.Refresh()
		}
		return m, nil

	case constants.SettingsSavedMsg:
		return m, nil

	case constants.SettingsErrorMsg:
		m.ctx.SetStatus(msg.Err.Error(), true)
		return m, nil

	case constants.ClearStatusMsg:
		m.ctx.ClearStatus()
		return m, nil
	}

	// Update components
	var cmd tea.Cmd

	m.header, cmd = m.header.Update(msg)
	cmds = append(cmds, cmd)

	m.sidebar, cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// LaunchResultMsg is sent when a launch completes.
type LaunchResultMsg struct {
	Sessions  int
	Worktrees int
	Err       error
}

func (m *Model) nextView() {
	switch m.ctx.View {
	case constants.WorktreesView:
		m.ctx.SetView(constants.LaunchView)
	case constants.LaunchView:
		m.ctx.SetView(constants.SettingsView)
	case constants.SettingsView:
		m.ctx.SetView(constants.PresetsView)
	case constants.PresetsView:
		m.ctx.SetView(constants.WorktreesView)
	}
	m.syncProgramContext()
}

func (m *Model) prevView() {
	switch m.ctx.View {
	case constants.WorktreesView:
		m.ctx.SetView(constants.PresetsView)
	case constants.PresetsView:
		m.ctx.SetView(constants.SettingsView)
	case constants.SettingsView:
		m.ctx.SetView(constants.LaunchView)
	case constants.LaunchView:
		m.ctx.SetView(constants.WorktreesView)
	}
	m.syncProgramContext()
}

func (m Model) doLaunch(req launch.LaunchRequestMsg) tea.Cmd {
	return func() tea.Msg {
		// Get preset
		var preset config.Preset
		if m.ctx.PresetConfig != nil {
			if p, ok := m.ctx.PresetConfig.GetPreset(req.Preset); ok {
				preset = p
			}
		}

		if len(preset) == 0 {
			// Default preset with a basic agent
			preset = config.Preset{{Agent: "claude"}}
		}

		// Launch
		result := launcher.Launch(launcher.Options{
			Repo:       req.Repo,
			Name:       req.Name,
			Prompt:     req.Prompt,
			PresetName: req.Preset,
			Multiplier: 1,
			DataDir:    m.ctx.DataDir,
			Preset:     preset,
		})

		if result.Error != nil {
			return LaunchResultMsg{Err: result.Error}
		}

		return LaunchResultMsg{
			Sessions:  len(result.Sessions),
			Worktrees: result.TerminalWindowCount,
		}
	}
}

func (m Model) doFocusWorktree(wt *worktrees.WorktreeItem) tea.Cmd {
	return func() tea.Msg {
		// Try to focus by path first
		found, err := terminal.FocusWorktreeWindow(wt.Path)
		if err == nil && found {
			return LaunchResultMsg{
				Sessions:  0,
				Worktrees: 0,
			}
		}

		// Fallback: try to focus by branch name
		if wt.Branch != "" {
			found, err = terminal.AlternativeFocusWorktreeWindow(wt.Branch)
			if err == nil && found {
				return LaunchResultMsg{
					Sessions:  0,
					Worktrees: 0,
				}
			}
		}

		// No existing window found
		return LaunchResultMsg{
			Err: fmt.Errorf("no existing iTerm window found for this worktree"),
		}
	}
}

func (m Model) doReopenWorktree(wt *worktrees.WorktreeItem) tea.Cmd {
	return func() tea.Msg {
		// Get preset from config
		var preset config.Preset
		if m.ctx.PresetConfig != nil && wt.PresetName != "" {
			if p, ok := m.ctx.PresetConfig.GetPreset(wt.PresetName); ok {
				preset = p
			}
		}

		// Fallback: use agents from metadata
		if len(preset) == 0 && len(wt.Agents) > 0 {
			preset = config.Preset{{Agent: wt.Agents[0]}}
		}

		if len(preset) == 0 {
			return LaunchResultMsg{Err: fmt.Errorf("no preset or agent information available")}
		}

		// Build sessions for this worktree
		var sessions []terminal.SessionInfo
		for _, agent := range wt.Agents {
			sessions = append(sessions, terminal.SessionInfo{
				Path:   wt.Path,
				Branch: wt.Branch,
				Agent:  agent,
			})
		}

		// Add command sessions if in preset
		for _, w := range preset {
			for _, cmd := range w.Commands {
				r, g, b, _ := config.ParseHexColor(cmd.Color)
				sessions = append(sessions, terminal.SessionInfo{
					IsCustomCommand: true,
					Command:         cmd.Command,
					Title:           cmd.GetTitle(),
					ColorR:          r,
					ColorG:          g,
					ColorB:          b,
					WorktreePath:    wt.Path,
					WorktreeBranch:  wt.Branch,
				})
			}
		}

		if len(sessions) == 0 {
			return LaunchResultMsg{Err: fmt.Errorf("no sessions to launch")}
		}

		// Connect to iTerm2 and launch sessions
		mgr, err := terminal.NewManager("Orchestrate")
		if err != nil {
			return LaunchResultMsg{Err: fmt.Errorf("failed to connect to iTerm2: %w", err)}
		}
		defer mgr.Close()

		windowCount, err := mgr.LaunchSessions(sessions, "")
		if err != nil {
			return LaunchResultMsg{Err: err}
		}

		return LaunchResultMsg{
			Sessions:  len(sessions),
			Worktrees: windowCount,
		}
	}
}

func (m Model) doDeleteWorktree(wt *worktrees.WorktreeItem) tea.Cmd {
	return func() tea.Msg {
		// Delete the worktree directory
		err := os.RemoveAll(wt.Path)
		if err != nil {
			return worktrees.WorktreeDeletedMsg{
				Path: wt.Path,
				Err:  fmt.Errorf("failed to delete worktree directory: %w", err),
			}
		}

		return worktrees.WorktreeDeletedMsg{
			Path: wt.Path,
			Err:  nil,
		}
	}
}

func (m Model) renderWorktreeDetails(wt *worktrees.WorktreeItem) string {
	var lines []string

	// Header
	lines = append(lines, m.ctx.Styles.Common.MainTextStyle.Render(wt.Name))
	lines = append(lines, "")

	// Session info if available
	if wt.HasMeta {
		lines = append(lines, m.ctx.Styles.Common.SuccessStyle.Render(
			constants.SuccessIcon+" Session metadata available"))
		lines = append(lines, "")
	}

	// Repo
	if wt.Repo != "" {
		lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render("Repo:"))
		lines = append(lines, m.ctx.Styles.Settings.Value.Render("  "+wt.Repo))
		lines = append(lines, "")
	}

	// Branch
	if wt.Branch != "" {
		lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render("Branch:"))
		lines = append(lines, m.ctx.Styles.Settings.Value.Render("  "+wt.Branch))
		lines = append(lines, "")
	}

	// Changes summary
	if wt.Adds > 0 || wt.Deletes > 0 {
		lines = append(lines, m.ctx.Styles.Settings.Category.Render("Changes:"))
		lines = append(lines, "")
		addStr := lipgloss.NewStyle().Foreground(m.ctx.Theme.SuccessText).Render(fmt.Sprintf("+%d", wt.Adds))
		delStr := lipgloss.NewStyle().Foreground(m.ctx.Theme.ErrorText).Render(fmt.Sprintf("-%d", wt.Deletes))
		lines = append(lines, fmt.Sprintf("  %s  %s", addStr, delStr))
		lines = append(lines, "")

		// File-level changes
		if len(wt.FileStats) > 0 {
			lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render("Files changed:"))
			maxFiles := 10
			for i, fs := range wt.FileStats {
				if i >= maxFiles {
					remaining := len(wt.FileStats) - maxFiles
					lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render(
						fmt.Sprintf("  ... and %d more files", remaining)))
					break
				}
				addStr := lipgloss.NewStyle().Foreground(m.ctx.Theme.SuccessText).Render(fmt.Sprintf("+%d", fs.Adds))
				delStr := lipgloss.NewStyle().Foreground(m.ctx.Theme.ErrorText).Render(fmt.Sprintf("-%d", fs.Deletes))
				fileName := fs.Path
				if len(fileName) > 30 {
					fileName = "..." + fileName[len(fileName)-27:]
				}
				lines = append(lines, fmt.Sprintf("  %s %s  %s", addStr, delStr, fileName))
			}
			lines = append(lines, "")
		}
	}

	// Preset and Agents
	if wt.PresetName != "" {
		lines = append(lines, m.ctx.Styles.Settings.Category.Render("Configuration:"))
		lines = append(lines, "")
		lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render("Preset:"))
		lines = append(lines, m.ctx.Styles.Settings.Value.Render("  "+wt.PresetName))
	}

	if len(wt.Agents) > 0 {
		lines = append(lines, "")
		lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render("Agents:"))
		for _, agent := range wt.Agents {
			lines = append(lines, m.ctx.Styles.Settings.Value.Render("  "+constants.AgentIcon+" "+agent))
		}
	}

	// Prompt (if available)
	if wt.Prompt != "" {
		lines = append(lines, "")
		lines = append(lines, m.ctx.Styles.Settings.Category.Render("Prompt:"))
		lines = append(lines, "")
		// Truncate long prompts
		prompt := wt.Prompt
		if len(prompt) > 200 {
			prompt = prompt[:197] + "..."
		}
		// Make prompt more visible with MainTextStyle and add a border
		promptStyle := lipgloss.NewStyle().
			Foreground(m.ctx.Theme.PrimaryText).
			Background(m.ctx.Theme.FaintBorder).
			Padding(1).
			Width(constants.SidebarWidth - 8)
		lines = append(lines, promptStyle.Render(prompt))
	}

	lines = append(lines, "")
	lines = append(lines, m.ctx.Styles.Settings.Category.Render("Details:"))
	lines = append(lines, "")
	lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render("Path:"))
	lines = append(lines, m.ctx.Styles.Settings.Value.Render("  "+wt.Path))
	lines = append(lines, "")
	lines = append(lines, m.ctx.Styles.Common.FaintTextStyle.Render("Last Commit:"))
	lines = append(lines, m.ctx.Styles.Settings.Value.Render("  "+wt.LastCommit))

	return strings.Join(lines, "\n")
}

// View renders the entire UI.
func (m Model) View() string {
	if !m.ready {
		return lipgloss.Place(
			m.ctx.ScreenWidth,
			m.ctx.ScreenHeight,
			lipgloss.Center,
			lipgloss.Center,
			m.ctx.Styles.Common.FaintTextStyle.Render("Loading..."),
		)
	}

	// Build layout
	headerView := m.header.View()
	contentView := m.renderContent()

	// Join content and sidebar horizontally
	mainArea := contentView
	if m.sidebar.IsOpen {
		sidebarView := m.sidebar.View()
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, contentView, sidebarView)
	}

	// Join all sections vertically and enforce total screen size to prevent scrolling
	return lipgloss.NewStyle().
		Width(m.ctx.ScreenWidth).
		Height(m.ctx.ScreenHeight).
		MaxWidth(m.ctx.ScreenWidth).
		MaxHeight(m.ctx.ScreenHeight).
		Background(m.ctx.Theme.Background).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			headerView,
			mainArea,
		))
}

func (m Model) renderContent() string {
	var content string

	switch m.ctx.View {
	case constants.WorktreesView:
		content = m.worktreeList.View()
	case constants.LaunchView:
		content = m.launchForm.View()
	case constants.SettingsView:
		content = m.settingsForm.View()
	case constants.PresetsView:
		content = m.presetsInfo.View()
	default:
		content = "Unknown view"
	}

	return content
}

func (m *Model) syncComponentDimensions() {
	dims := constants.Dimensions{
		Width:  m.ctx.MainContentWidth,
		Height: m.ctx.MainContentHeight,
	}

	m.settingsForm.SetDimensions(dims)
	m.presetsInfo.SetDimensions(dims)
	m.worktreeList.SetDimensions(dims)
	m.launchForm.SetDimensions(dims)
}

func (m *Model) syncProgramContext() {
	m.header.UpdateProgramContext(m.ctx)
	m.sidebar.UpdateProgramContext(m.ctx)
	m.settingsForm.UpdateProgramContext(m.ctx)
	m.presetsInfo.UpdateProgramContext(m.ctx)
	m.worktreeList.UpdateProgramContext(m.ctx)
	m.launchForm.UpdateProgramContext(m.ctx)
}

func (m Model) saveSettings() tea.Cmd {
	return func() tea.Msg {
		settings := m.settingsForm.GetSettings()
		if settings == nil {
			return constants.SettingsErrorMsg{Err: nil}
		}

		err := config.SaveAppSettings(m.ctx.DataDir, settings)
		if err != nil {
			return constants.SettingsErrorMsg{Err: err}
		}
		return constants.SettingsSavedMsg{}
	}
}

// Run starts the TUI application.
func Run(dataDir string, appSettings *config.AppSettings, presetConfig *config.Config) error {
	model := NewModel(dataDir, appSettings, presetConfig)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
