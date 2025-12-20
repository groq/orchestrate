// Package worktrees provides the worktrees dashboard component.
package worktrees

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"orchestrate/config"
	"orchestrate/git_utils"
	"orchestrate/internal/tui/constants"
	"orchestrate/internal/tui/context"
	"orchestrate/internal/tui/keys"
	"orchestrate/internal/tui/theme"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// WorktreeItem represents a worktree for display.
type WorktreeItem struct {
	Path       string
	Name       string // Directory name
	Branch     string
	Repo       string
	LastCommit string
	CreatedAt  time.Time
	Prompt     string
	PresetName string
	Agents     []string
	HasMeta    bool
	Adds       int
	Deletes    int
	FileStats  []git_utils.FileStats
}

// Model represents the worktrees component.
type Model struct {
	ctx              *context.ProgramContext
	worktrees        []WorktreeItem
	selected         int
	loading          bool
	spinner          spinner.Model
	err              error
	dimensions       constants.Dimensions
	worktreesDir     string
	confirmingDelete bool
	deleteTarget     *WorktreeItem
}

// WorktreesLoadedMsg is sent when worktrees are loaded.
type WorktreesLoadedMsg struct {
	Worktrees []WorktreeItem
	Err       error
}

// WorktreeDetailsMsg is sent when a worktree is selected for detailed view.
type WorktreeDetailsMsg struct {
	Worktree *WorktreeItem
}

// OpenWorktreeMsg is sent when a worktree should be opened in iTerm (new window).
type OpenWorktreeMsg struct {
	Worktree *WorktreeItem
}

// FocusWorktreeMsg is sent when an existing worktree iTerm window should be focused.
type FocusWorktreeMsg struct {
	Worktree *WorktreeItem
}

// DeleteWorktreeMsg is sent when a worktree should be deleted.
type DeleteWorktreeMsg struct {
	Worktree *WorktreeItem
}

// WorktreeDeletedMsg is sent when a worktree has been deleted.
type WorktreeDeletedMsg struct {
	Path string
	Err  error
}

// New creates a new worktrees model.
func New(ctx *context.ProgramContext) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.LogoColor)

	// Default worktrees directory
	worktreesDir := ""
	if ctx != nil && ctx.DataDir != "" {
		worktreesDir = filepath.Join(ctx.DataDir, "worktrees")
	}

	return Model{
		ctx:          ctx,
		loading:      false,
		spinner:      s,
		selected:     0,
		worktreesDir: worktreesDir,
	}
}

// Init initializes the model and starts loading.
func (m Model) Init() tea.Cmd {
	return m.Refresh()
}

// Refresh reloads the worktrees.
func (m *Model) Refresh() tea.Cmd {
	m.loading = true
	return tea.Batch(m.spinner.Tick, m.loadWorktrees())
}

func (m Model) loadWorktrees() tea.Cmd {
	return func() tea.Msg {
		if m.worktreesDir == "" {
			return WorktreesLoadedMsg{Worktrees: nil, Err: nil}
		}

		// Check if worktrees directory exists
		if _, err := os.Stat(m.worktreesDir); os.IsNotExist(err) {
			return WorktreesLoadedMsg{Worktrees: nil, Err: nil}
		}

		// Read all directories in worktrees folder
		entries, err := os.ReadDir(m.worktreesDir)
		if err != nil {
			return WorktreesLoadedMsg{Err: err}
		}

		var items []WorktreeItem
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			wtPath := filepath.Join(m.worktreesDir, entry.Name())

			// Check if it's a git directory
			gitDir := filepath.Join(wtPath, ".git")
			if _, err := os.Stat(gitDir); os.IsNotExist(err) {
				continue
			}

			item := WorktreeItem{
				Path: wtPath,
				Name: entry.Name(),
			}

			// Get branch name
			if branch, err := getGitBranch(wtPath); err == nil {
				item.Branch = branch
			}

			// Get last commit time
			if lastCommit, err := getLastCommitTime(wtPath); err == nil {
				item.LastCommit = lastCommit
			}

			// Get repo origin URL
			if repo, err := getGitRemote(wtPath); err == nil {
				item.Repo = repo
			}

			// Get uncommitted changes stats
			if a, d, err := git_utils.GetStatusStats(wtPath); err == nil {
				item.Adds = a
				item.Deletes = d
			}

			// Get detailed file stats
			if fileStats, err := git_utils.GetDetailedStatusStats(wtPath); err == nil {
				item.FileStats = fileStats
			}

			// Try to load session metadata
			if meta, err := config.LoadSessionMetadata(wtPath); err == nil {
				item.HasMeta = true
				item.CreatedAt = meta.CreatedAt
				item.Prompt = meta.Prompt
				item.PresetName = meta.PresetName
				item.Agents = meta.Agents
				if item.Repo == "" {
					item.Repo = meta.Repo
				}
			} else {
				// Try to get creation time from directory
				if info, err := entry.Info(); err == nil {
					item.CreatedAt = info.ModTime()
				}
			}

			items = append(items, item)
		}

		return WorktreesLoadedMsg{Worktrees: items}
	}
}

func getGitBranch(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getLastCommitTime(path string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%cr")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getGitRemote(path string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Parse GitHub URL to short form
	url := strings.TrimSpace(string(out))
	url = strings.TrimSuffix(url, ".git")
	if strings.Contains(url, "github.com") {
		parts := strings.Split(url, "github.com")
		if len(parts) > 1 {
			return strings.TrimPrefix(parts[1], "/"), nil
		}
	}
	return url, nil
}

// IsAtTop returns true if the first item is selected.
func (m Model) IsAtTop() bool {
	return m.selected == 0
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case WorktreesLoadedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.worktrees = msg.Worktrees
		m.err = nil
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// If confirming delete, only handle y/n
		if m.confirmingDelete {
			switch msg.String() {
			case "y", "Y":
				// Confirm delete
				if m.deleteTarget != nil {
					target := m.deleteTarget
					m.confirmingDelete = false
					m.deleteTarget = nil
					return m, func() tea.Msg {
						return DeleteWorktreeMsg{Worktree: target}
					}
				}
			case "n", "N", "esc":
				// Cancel delete
				m.confirmingDelete = false
				m.deleteTarget = nil
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.worktrees)-1 {
				m.selected++
			}
		case "g":
			m.selected = 0
		case "G":
			if len(m.worktrees) > 0 {
				m.selected = len(m.worktrees) - 1
			}
		case "enter":
			// Focus existing iTerm window for this worktree
			if wt := m.SelectedWorktree(); wt != nil {
				return m, func() tea.Msg {
					return FocusWorktreeMsg{Worktree: wt}
				}
			}
		case "d":
			// Show details in sidebar
			if wt := m.SelectedWorktree(); wt != nil {
				return m, func() tea.Msg {
					return WorktreeDetailsMsg{Worktree: wt}
				}
			}
		case "o":
			// Open worktree in iTerm with same preset
			if wt := m.SelectedWorktree(); wt != nil {
				return m, func() tea.Msg {
					return OpenWorktreeMsg{Worktree: wt}
				}
			}
		case "x", "delete":
			// Show delete confirmation
			if wt := m.SelectedWorktree(); wt != nil {
				m.confirmingDelete = true
				m.deleteTarget = wt
			}
		}

		if key.Matches(msg, keys.Keys.Refresh) {
			return m, m.Refresh()
		}

	case WorktreeDeletedMsg:
		m.loading = false
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		// Refresh the worktree list after deletion
		return m, m.Refresh()
	}

	return m, nil
}

// View renders the worktrees view.
func (m Model) View() string {
	if m.ctx == nil {
		return ""
	}

	var content strings.Builder

	// Header section with title and count
	count := len(m.worktrees)
	title := m.ctx.Styles.Settings.Category.Render(
		fmt.Sprintf(" %s Worktrees (%d)", constants.WorktreeIcon, count))
	content.WriteString(title)
	content.WriteString("\n\n")

	if m.loading {
		content.WriteString(lipgloss.NewStyle().
			Foreground(theme.LogoColor).
			Render(m.spinner.View() + " Loading worktrees..."))
		return m.wrapContent(content.String())
	}

	if m.err != nil {
		content.WriteString(m.ctx.Styles.Common.ErrorStyle.Render(
			fmt.Sprintf("%s Error: %v", constants.FailureIcon, m.err)))
		return m.wrapContent(content.String())
	}

	if len(m.worktrees) == 0 {
		content.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render(
			"No worktrees found in " + m.worktreesDir))
		content.WriteString("\n\n")
		content.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render(
			"Create worktrees with:"))
		content.WriteString("\n")
		content.WriteString(m.ctx.Styles.Settings.Value.Render(
			"  orchestrate --repo owner/repo --name feature --prompt \"...\""))
		return m.wrapContent(content.String())
	}

	// Table header
	headerStyle := m.ctx.Styles.Table.TitleCellStyle
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerStyle.Width(4).Render(""),
		headerStyle.Width(35).Render("Name"),
		headerStyle.Width(30).Render("Repo"),
		headerStyle.Width(18).Render("Changes"),
		headerStyle.Width(15).Render("Agent"),
		headerStyle.Width(18).Render("Last Commit"),
	)
	content.WriteString(header)
	content.WriteString("\n")

	// Separator
	sepStyle := lipgloss.NewStyle().Foreground(m.ctx.Theme.FaintBorder)
	content.WriteString(sepStyle.Render(strings.Repeat("─", min(m.dimensions.Width-4, 120))))
	content.WriteString("\n")

	// Worktree rows (limit visible rows based on height)
	maxRows := m.dimensions.Height - 12
	if maxRows < 5 {
		maxRows = 5
	}

	startIdx := 0
	if m.selected >= maxRows {
		startIdx = m.selected - maxRows + 1
	}

	for i := startIdx; i < len(m.worktrees) && i < startIdx+maxRows; i++ {
		row := m.renderWorktreeRow(i, m.worktrees[i])
		content.WriteString(row)
		content.WriteString("\n")
	}

	// Scroll indicator if needed
	if len(m.worktrees) > maxRows {
		scrollInfo := fmt.Sprintf(" %d-%d of %d ",
			startIdx+1, min(startIdx+maxRows, len(m.worktrees)), len(m.worktrees))
		content.WriteString(m.ctx.Styles.Common.FaintTextStyle.Render(scrollInfo))
		content.WriteString("\n")
	}

	// Footer with actions
	content.WriteString("\n")
	content.WriteString(m.renderActions())

	output := m.wrapContent(content.String())

	// Show confirmation dialog if deleting
	if m.confirmingDelete && m.deleteTarget != nil {
		output = m.renderWithConfirmDialog(output, m.deleteTarget)
	}

	return output
}

func (m Model) renderWorktreeRow(index int, wt WorktreeItem) string {
	isSelected := index == m.selected

	// Status icon
	statusIcon := constants.BranchIcon
	statusStyle := lipgloss.NewStyle().Foreground(theme.LogoColor)
	if wt.HasMeta {
		statusIcon = constants.SuccessIcon
		statusStyle = m.ctx.Styles.Status.Active
	}

	// Name (shortened if needed)
	name := wt.Name
	if len(name) > 34 {
		name = name[:31] + "..."
	}

	// Repo (shortened)
	repo := wt.Repo
	if repo == "" {
		repo = "-"
	} else if len(repo) > 28 {
		repo = "..." + repo[len(repo)-25:]
	}

	// Stats badge
	statsBadge := ""
	if wt.Adds > 0 || wt.Deletes > 0 {
		addStr := ""
		if wt.Adds > 0 {
			addStr = lipgloss.NewStyle().Foreground(m.ctx.Theme.SuccessText).Render(fmt.Sprintf("+%d", wt.Adds))
		}
		delStr := ""
		if wt.Deletes > 0 {
			delStr = lipgloss.NewStyle().Foreground(m.ctx.Theme.ErrorText).Render(fmt.Sprintf("-%d", wt.Deletes))
		}
		statsBadge = fmt.Sprintf("[%s%s]", addStr, delStr)
	}

	// Agent badge
	agentBadge := ""
	if len(wt.Agents) > 0 {
		agentName := wt.Agents[0]
		agentColor := theme.GetAgentColor(agentName)
		agentBadge = lipgloss.NewStyle().
			Foreground(agentColor).
			Background(m.ctx.Theme.FaintBorder).
			Padding(0, 1).
			Render(agentName)
	}

	// Last commit
	lastCommit := wt.LastCommit
	if lastCommit == "" {
		lastCommit = "-"
	}

	// Build row with appropriate styling
	cellStyle := m.ctx.Styles.Table.CellStyle
	if isSelected {
		cellStyle = m.ctx.Styles.Table.SelectedCellStyle
	}

	// Cursor
	cursor := "  "
	if isSelected {
		cursor = m.ctx.Styles.Settings.Cursor.Render(constants.Cursor + " ")
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		cellStyle.Width(4).Render(cursor+statusStyle.Render(statusIcon)),
		cellStyle.Width(35).Render(name),
		cellStyle.Width(30).Render(repo),
		cellStyle.Width(18).Render(statsBadge),
		cellStyle.Width(15).Render(agentBadge),
		cellStyle.Width(18).Render(lastCommit),
	)

	return row
}

func (m Model) renderActions() string {
	actions := []struct {
		key  string
		desc string
	}{
		{"↑/↓", "navigate"},
		{"enter", "focus"},
		{"o", "open"},
		{"d", "details"},
		{"x", "delete"},
		{"ctrl+r", "refresh"},
	}

	var parts []string
	for _, a := range actions {
		keyStyle := lipgloss.NewStyle().Foreground(m.ctx.Theme.PrimaryText).Bold(true)
		descStyle := m.ctx.Styles.Common.FaintTextStyle
		parts = append(parts, keyStyle.Render(a.key)+" "+descStyle.Render(a.desc))
	}

	return lipgloss.NewStyle().
		Foreground(m.ctx.Theme.FaintText).
		Render(strings.Join(parts, "  |  "))
}

func (m Model) wrapContent(content string) string {
	return lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.dimensions.Width).
		Height(m.dimensions.Height).
		Render(content)
}

func (m Model) renderWithConfirmDialog(background string, wt *WorktreeItem) string {
	// Create confirmation dialog
	dialogWidth := 60
	dialogHeight := 10

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.ctx.Theme.ErrorText).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	messageStyle := lipgloss.NewStyle().
		Foreground(m.ctx.Theme.PrimaryText).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	infoStyle := lipgloss.NewStyle().
		Foreground(m.ctx.Theme.FaintText).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	promptStyle := lipgloss.NewStyle().
		Foreground(m.ctx.Theme.PrimaryText).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	var dialogContent strings.Builder
	dialogContent.WriteString(titleStyle.Render(constants.WarningIcon + " Delete Worktree"))
	dialogContent.WriteString("\n\n")
	dialogContent.WriteString(messageStyle.Render("Are you sure you want to delete this worktree?"))
	dialogContent.WriteString("\n\n")
	dialogContent.WriteString(infoStyle.Render("Name: " + wt.Name))
	dialogContent.WriteString("\n")
	dialogContent.WriteString(infoStyle.Render("Path: " + wt.Path))
	dialogContent.WriteString("\n\n")
	dialogContent.WriteString(promptStyle.Render("Press Y to confirm, N to cancel"))

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.ctx.Theme.ErrorText).
		Background(m.ctx.Theme.Background).
		Width(dialogWidth).
		Height(dialogHeight).
		Padding(1, 2).
		Render(dialogContent.String())

	// Overlay dialog on background
	return lipgloss.Place(
		m.dimensions.Width,
		m.dimensions.Height,
		lipgloss.Center,
		lipgloss.Center,
		dialogBox,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#000000")),
	)
}

// SetDimensions sets the component dimensions.
func (m *Model) SetDimensions(dims constants.Dimensions) {
	m.dimensions = dims
}

// UpdateProgramContext updates the context.
func (m *Model) UpdateProgramContext(ctx *context.ProgramContext) {
	m.ctx = ctx
	if ctx != nil && ctx.DataDir != "" {
		m.worktreesDir = filepath.Join(ctx.DataDir, "worktrees")
	}
}

// SelectedWorktree returns the currently selected worktree.
func (m Model) SelectedWorktree() *WorktreeItem {
	if m.selected >= 0 && m.selected < len(m.worktrees) {
		return &m.worktrees[m.selected]
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
