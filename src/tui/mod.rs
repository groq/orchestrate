use crate::config::appsettings::{self, AppSettings};
use crate::config::preset::{self, Config as PresetConfig};
use crate::config::session::{self};
use crate::git;
use crate::launcher;
use crate::terminal;
use crate::util;
use anyhow::Result;
use chrono::{DateTime, Local};
use crossterm::cursor::{Hide, Show};
use crossterm::event::{self, Event, KeyCode, KeyEvent, KeyModifiers};
use crossterm::terminal::{
    disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen,
};
use crossterm::ExecutableCommand;
use ratatui::backend::CrosstermBackend;
use ratatui::layout::{Alignment, Constraint, Direction, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{
    Block, BorderType, Borders, Cell, Clear, List, ListItem, Paragraph, Row, Table, Wrap,
};
use ratatui::Terminal;
use std::fs;
use std::path::PathBuf;
use std::time::{Duration, Instant};

type Frame<'a> = ratatui::Frame<'a>;

// Modern color palette - warm orange theme
const ACCENT: Color = Color::Rgb(255, 140, 0); // Orange accent
const ACCENT_DIM: Color = Color::Rgb(180, 100, 0); // Dimmed orange
const ACCENT_HIGHLIGHT_BG: Color = Color::Rgb(50, 40, 30); // Subtle warm tint for selected rows
const STATUS_ACTIVE_BG: Color = Color::Rgb(40, 160, 80); // Green for active
const STATUS_ACTIVE_FG: Color = Color::Rgb(180, 255, 200); // Light green text
const STATUS_IDLE_BG: Color = Color::Rgb(180, 60, 60); // Red for idle
const STATUS_IDLE_FG: Color = Color::Rgb(255, 200, 200); // Light red text
const ZEBRA_BG: Color = Color::Rgb(28, 26, 24); // Darker, warmer zebra
const LIGHT_BORDER: Color = Color::Rgb(80, 75, 70); // Subtle border
const DIM_TEXT: Color = Color::Rgb(120, 110, 100); // Dimmed text
const SURFACE_BG: Color = Color::Rgb(22, 20, 18); // Card background
const HEADER_BG: Color = Color::Rgb(30, 28, 25); // Header background
const SUCCESS_COLOR: Color = Color::Rgb(100, 220, 150); // Green for additions
const ERROR_COLOR: Color = Color::Rgb(240, 100, 100); // Red for deletions/errors
const WARNING_COLOR: Color = Color::Rgb(240, 180, 80); // Yellow for warnings

pub fn run(
    data_dir: PathBuf,
    app_settings: AppSettings,
    preset_config: Option<PresetConfig>,
    preset_error: Option<String>,
) -> Result<()> {
    enable_raw_mode()?;
    let mut stdout = std::io::stdout();
    stdout.execute(EnterAlternateScreen)?;
    stdout.execute(Hide)?;

    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let mut app = App::new(data_dir, app_settings, preset_config, preset_error);
    app.refresh_worktrees()?;

    let tick_rate = Duration::from_millis(250);
    let mut last_tick = Instant::now();

    loop {
        terminal.draw(|f| {
            app.draw(f);
        })?;

        let timeout = tick_rate
            .checked_sub(last_tick.elapsed())
            .unwrap_or_else(|| Duration::from_secs(0));

        let mut handled_event = false;
        if event::poll(timeout)? {
            match event::read()? {
                Event::Key(key) => {
                    handled_event = app.on_key(key)?;
                }
                Event::Resize(_, _) => {
                    // redraw on next loop
                }
                _ => {}
            }
        }

        if last_tick.elapsed() >= tick_rate {
            last_tick = Instant::now();
            app.on_tick();
        }

        if app.should_quit {
            break;
        }

        // If we consumed a key event, continue loop; otherwise keep drawing
        if handled_event {
            continue;
        }
    }

    disable_raw_mode()?;
    let mut stdout = std::io::stdout();
    stdout.execute(Show)?;
    stdout.execute(LeaveAlternateScreen)?;
    Ok(())
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum View {
    Worktrees,
    Launch,
    Settings,
    Presets,
}

/// Actions that can be triggered from the worktrees view
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum WorktreeAction {
    /// Navigate up in the list
    NavigateUp,
    /// Navigate down in the list
    NavigateDown,
    /// Jump to the first item
    JumpToFirst,
    /// Jump to the last item
    JumpToLast,
    /// Toggle details sidebar
    ToggleDetails,
    /// Expand/collapse prompt
    TogglePromptExpanded,
    /// Open/reopen the selected worktree
    Open,
    /// Focus or reopen the selected worktree
    FocusOrOpen,
    /// Initiate delete (shows confirmation)
    InitiateDelete,
    /// Refresh the worktree list
    Refresh,
    /// Toggle sidebar
    ToggleSidebar,
    /// No action for this key
    None,
}

/// Actions for the delete confirmation dialog
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum ConfirmDeleteAction {
    /// Confirm deletion
    Confirm,
    /// Cancel deletion
    Cancel,
    /// No action (key not handled)
    None,
}

/// Maps a key event to a worktree action (when not in confirmation dialog)
fn map_worktree_key(key: KeyEvent) -> WorktreeAction {
    match key.code {
        KeyCode::Char('p') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            WorktreeAction::ToggleSidebar
        }
        KeyCode::Char('r') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            WorktreeAction::Refresh
        }
        KeyCode::Up | KeyCode::Char('k') => WorktreeAction::NavigateUp,
        KeyCode::Down | KeyCode::Char('j') => WorktreeAction::NavigateDown,
        KeyCode::Char('g') => WorktreeAction::JumpToFirst,
        KeyCode::Char('G') => WorktreeAction::JumpToLast,
        KeyCode::Char('d') => WorktreeAction::ToggleDetails,
        KeyCode::Char('e') => WorktreeAction::TogglePromptExpanded,
        KeyCode::Char('o') => WorktreeAction::Open,
        KeyCode::Enter => WorktreeAction::FocusOrOpen,
        KeyCode::Char('x') | KeyCode::Delete | KeyCode::Backspace => WorktreeAction::InitiateDelete,
        _ => WorktreeAction::None,
    }
}

/// Maps a key event to a confirmation dialog action
fn map_confirm_delete_key(key: KeyEvent) -> ConfirmDeleteAction {
    match key.code {
        KeyCode::Char('y') | KeyCode::Char('Y') => ConfirmDeleteAction::Confirm,
        KeyCode::Char('n') | KeyCode::Esc => ConfirmDeleteAction::Cancel,
        _ => ConfirmDeleteAction::None,
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum LaunchField {
    Repo,
    Name,
    Prompt,
    Preset,
    Launch,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SettingsField {
    TerminalType,
    Maximize,
    DefaultPreset,
    AutoClean,
    RetentionDays,
}

#[derive(Debug, Clone)]
struct LaunchForm {
    repo: String,
    name: String,
    prompt: String,
    preset_names: Vec<String>,
    preset_idx: usize,
    focused: LaunchField,
}

impl LaunchForm {
    fn new(preset_names: Vec<String>, default_preset: String) -> Self {
        let mut preset_idx = 0;
        if !preset_names.is_empty() {
            if let Some(idx) = preset_names.iter().position(|p| p == &default_preset) {
                preset_idx = idx;
            }
        }
        LaunchForm {
            repo: String::new(),
            name: String::new(),
            prompt: String::new(),
            preset_names,
            preset_idx,
            focused: LaunchField::Repo,
        }
    }

    fn next_field(&mut self) {
        self.focused = match self.focused {
            LaunchField::Repo => LaunchField::Name,
            LaunchField::Name => LaunchField::Prompt,
            LaunchField::Prompt => LaunchField::Preset,
            LaunchField::Preset => LaunchField::Launch,
            LaunchField::Launch => LaunchField::Repo,
        };
    }

    fn prev_field(&mut self) {
        self.focused = match self.focused {
            LaunchField::Repo => LaunchField::Launch,
            LaunchField::Name => LaunchField::Repo,
            LaunchField::Prompt => LaunchField::Name,
            LaunchField::Preset => LaunchField::Prompt,
            LaunchField::Launch => LaunchField::Preset,
        };
    }

    fn next_preset(&mut self) {
        if self.preset_names.is_empty() {
            return;
        }
        self.preset_idx = (self.preset_idx + 1) % self.preset_names.len();
    }

    fn prev_preset(&mut self) {
        if self.preset_names.is_empty() {
            return;
        }
        if self.preset_idx == 0 {
            self.preset_idx = self.preset_names.len() - 1;
        } else {
            self.preset_idx -= 1;
        }
    }

    fn preset(&self) -> String {
        if self.preset_names.is_empty() {
            "default".to_string()
        } else {
            self.preset_names[self.preset_idx].clone()
        }
    }

    fn submit(&self) -> Option<(String, String, String, String)> {
        let repo = self.repo.trim().to_string();
        let name = self.name.trim().to_string();
        let prompt = self.prompt.trim().to_string();
        if repo.is_empty() || name.is_empty() || prompt.is_empty() {
            return None;
        }
        Some((repo, name, prompt, self.preset()))
    }
}

#[derive(Debug, Clone)]
struct WorktreeItem {
    path: PathBuf,
    name: String,
    branch: String,
    repo: String,
    last_commit: String,
    created_at: Option<DateTime<Local>>,
    prompt: String,
    preset_name: String,
    agents: Vec<String>,
    has_meta: bool,
    adds: i32,
    deletes: i32,
    file_stats: Vec<git::FileStats>,
    recent_commits: Vec<String>,
    activity_log: Option<PathBuf>,
    activity_recent: Vec<String>,
    activity_active: bool,
}

impl WorktreeItem {
    fn display_agent(&self) -> String {
        self.agents
            .first()
            .cloned()
            .unwrap_or_else(|| "-".to_string())
    }
}

fn clean_log_line(line: String) -> String {
    // Strip ANSI escape sequences for display (keeps them in actual log files)
    let mut cleaned = String::new();
    let mut chars = line.chars().peekable();
    while let Some(ch) = chars.next() {
        if ch == '\x1b' {
            // Start of ANSI escape sequence
            if chars.peek() == Some(&'[') {
                chars.next(); // consume '['
                              // Skip until we hit a letter (the command character)
                while let Some(&c) = chars.peek() {
                    chars.next();
                    if c.is_ascii_alphabetic() || c == 'H' || c == 'K' || c == 'm' {
                        break;
                    }
                }
                continue;
            }
            // Also handle other escape sequences like \x1b]
            if chars.peek() == Some(&']') {
                chars.next();
                // OSC sequence - skip until BEL (\x07) or ST (\x1b\\)
                while let Some(c) = chars.next() {
                    if c == '\x07' {
                        break;
                    }
                    if c == '\x1b' && chars.peek() == Some(&'\\') {
                        chars.next();
                        break;
                    }
                }
                continue;
            }
            continue;
        }
        if ch == '\t' {
            cleaned.push_str("  ");
            continue;
        }
        if ch.is_control() {
            continue;
        }
        cleaned.push(ch);
    }
    if cleaned.len() > 180 {
        cleaned.truncate(177);
        cleaned.push('…');
    }
    cleaned
}

#[derive(Debug)]
struct SettingsForm {
    focused: SettingsField,
    app_settings: AppSettings,
}

impl SettingsForm {
    fn new(app_settings: AppSettings) -> Self {
        SettingsForm {
            focused: SettingsField::TerminalType,
            app_settings,
        }
    }

    fn next_field(&mut self) {
        self.focused = match self.focused {
            SettingsField::TerminalType => SettingsField::Maximize,
            SettingsField::Maximize => SettingsField::DefaultPreset,
            SettingsField::DefaultPreset => SettingsField::AutoClean,
            SettingsField::AutoClean => SettingsField::RetentionDays,
            SettingsField::RetentionDays => SettingsField::TerminalType,
        }
    }

    fn prev_field(&mut self) {
        self.focused = match self.focused {
            SettingsField::TerminalType => SettingsField::RetentionDays,
            SettingsField::Maximize => SettingsField::TerminalType,
            SettingsField::DefaultPreset => SettingsField::Maximize,
            SettingsField::AutoClean => SettingsField::DefaultPreset,
            SettingsField::RetentionDays => SettingsField::AutoClean,
        }
    }
}

struct App {
    data_dir: PathBuf,
    app_settings: AppSettings,
    preset_config: Option<PresetConfig>,
    view: View,
    sidebar_open: bool,
    status: Option<(String, bool)>,
    help_expanded: bool,
    actions_bar: bool,
    should_quit: bool,

    // Worktrees
    worktrees: Vec<WorktreeItem>,
    worktrees_loading: bool,
    selected_worktree: usize,
    show_details: bool,
    prompt_expanded: bool,
    confirming_delete: bool,
    last_refresh: Instant,

    // Launch
    launch_form: LaunchForm,

    // Settings
    settings_form: SettingsForm,
}

impl App {
    fn new(
        data_dir: PathBuf,
        app_settings: AppSettings,
        preset_config: Option<PresetConfig>,
        preset_error: Option<String>,
    ) -> Self {
        let preset_names = preset_config
            .as_ref()
            .map(|p| p.presets.keys().cloned().collect::<Vec<_>>())
            .unwrap_or_else(|| vec!["default".to_string()]);
        let default_preset = if let Some(cfg) = preset_config.as_ref() {
            preset::get_default_preset_name(cfg, &app_settings.session.default_preset)
        } else {
            app_settings.session.default_preset.clone()
        };
        let launch_form = LaunchForm::new(preset_names, default_preset);
        let settings_form = SettingsForm::new(app_settings.clone());

        // Show error as initial status if there was a config parse error
        let status = preset_error
            .as_ref()
            .map(|e| (format!("Config error: {}", e), true));

        App {
            data_dir,
            app_settings,
            preset_config,
            view: View::Worktrees,
            sidebar_open: false,
            status,
            help_expanded: false,
            actions_bar: false,
            should_quit: false,
            worktrees: Vec::new(),
            worktrees_loading: false,
            selected_worktree: 0,
            show_details: false,
            prompt_expanded: false,
            confirming_delete: false,
            last_refresh: Instant::now(),
            launch_form,
            settings_form,
        }
    }

    fn refresh_worktrees(&mut self) -> Result<()> {
        self.worktrees_loading = true;
        let dir = self.data_dir.join("worktrees");
        let mut items = Vec::new();
        if dir.exists() {
            for entry in fs::read_dir(&dir)? {
                let entry = entry?;
                if !entry.file_type()?.is_dir() {
                    continue;
                }
                let wt_path = entry.path();
                if !wt_path.join(".git").exists() {
                    continue;
                }
                let name = entry.file_name().to_string_lossy().to_string();
                let branch = git::current_branch(&wt_path).unwrap_or_else(|_| "-".to_string());
                let last_commit =
                    git::last_commit_time(&wt_path).unwrap_or_else(|_| "-".to_string());
                let repo_remote = git::remote_url(&wt_path).unwrap_or_default();
                let repo_short = if repo_remote.contains("github.com") {
                    repo_remote
                        .split("github.com/")
                        .nth(1)
                        .unwrap_or(&repo_remote)
                        .to_string()
                } else {
                    repo_remote.clone()
                };
                let (adds, deletes) = git::get_status_stats(&wt_path).unwrap_or((0, 0));
                let file_stats = git::get_detailed_status_stats(&wt_path).unwrap_or_default();
                let recent_commits = git::recent_commits(&wt_path, 3).unwrap_or_default();

                let activity_path = if !branch.is_empty() {
                    Some(
                        self.data_dir
                            .join("activity")
                            .join(format!("{}.log", branch)),
                    )
                } else {
                    None
                };
                let (activity_recent, activity_active) = if let Some(path) = &activity_path {
                    let lines = util::tail_lines(path, 6)
                        .unwrap_or_default()
                        .into_iter()
                        .map(clean_log_line)
                        .collect();
                    let active = util::modified_within(path, 10).unwrap_or(false);
                    (lines, active)
                } else {
                    (vec![], false)
                };

                let meta = session::load_session_metadata(&wt_path).ok();
                let (has_meta, created_at, prompt, preset_name, agents, repo) =
                    if let Some(m) = meta {
                        (
                            true,
                            Some(m.created_at.with_timezone(&Local)),
                            m.prompt,
                            m.preset_name,
                            m.agents,
                            if repo_short.is_empty() {
                                m.repo
                            } else {
                                repo_short.clone()
                            },
                        )
                    } else {
                        (
                            false,
                            None,
                            "".to_string(),
                            "".to_string(),
                            vec![],
                            repo_short.clone(),
                        )
                    };

                // Auto-clean stale worktrees if enabled
                if self.app_settings.session.auto_clean_worktrees {
                    let cutoff = chrono::Local::now()
                        - chrono::Duration::days(self.app_settings.session.worktree_retention_days);
                    let created_for_ret = created_at.unwrap_or_else(|| {
                        entry
                            .metadata()
                            .ok()
                            .and_then(|m| m.modified().ok())
                            .map(DateTime::<Local>::from)
                            .unwrap_or_else(Local::now)
                    });
                    if created_for_ret < cutoff {
                        let _ = std::fs::remove_dir_all(&wt_path);
                        continue;
                    }
                }

                items.push(WorktreeItem {
                    path: wt_path.clone(),
                    name,
                    branch,
                    repo,
                    last_commit,
                    created_at,
                    prompt,
                    preset_name,
                    agents,
                    has_meta,
                    adds,
                    deletes,
                    file_stats,
                    recent_commits,
                    activity_log: activity_path,
                    activity_recent,
                    activity_active,
                });
            }
        }
        items.sort_by(|a, b| b.created_at.cmp(&a.created_at));
        self.worktrees = items;
        if self.selected_worktree >= self.worktrees.len() {
            self.selected_worktree = self.worktrees.len().saturating_sub(1);
        }
        self.worktrees_loading = false;
        Ok(())
    }

    fn on_tick(&mut self) {
        // Refresh activity for all worktrees to keep status accurate
        for wt in &mut self.worktrees {
            if let Some(path) = &wt.activity_log {
                wt.activity_recent = util::tail_lines(path, 10)
                    .unwrap_or_default()
                    .into_iter()
                    .map(clean_log_line)
                    .collect();
                wt.activity_active = util::modified_within(path, 10).unwrap_or(false);
            }
        }
        // Periodic auto-refresh of worktrees
        if self.last_refresh.elapsed() >= Duration::from_secs(5) {
            let _ = self.refresh_worktrees();
            self.last_refresh = Instant::now();
        }
    }

    fn on_key(&mut self, key: KeyEvent) -> Result<bool> {
        if key.code == KeyCode::Char('c') && key.modifiers.contains(KeyModifiers::CONTROL) {
            self.should_quit = true;
            return Ok(true);
        }

        if key.code == KeyCode::Char('?') {
            self.actions_bar = !self.actions_bar;
            return Ok(true);
        }
        if self.actions_bar && key.code == KeyCode::Esc {
            self.actions_bar = false;
            return Ok(true);
        }

        if key.code == KeyCode::Tab && !key.modifiers.contains(KeyModifiers::SHIFT) {
            self.next_view();
            return Ok(true);
        }
        if key.code == KeyCode::Tab && key.modifiers.contains(KeyModifiers::SHIFT) {
            self.prev_view();
            return Ok(true);
        }

        match self.view {
            View::Worktrees => self.handle_worktrees_key(key),
            View::Launch => self.handle_launch_key(key),
            View::Settings => self.handle_settings_key(key),
            View::Presets => self.handle_presets_key(key),
        }
    }

    fn handle_worktrees_key(&mut self, key: KeyEvent) -> Result<bool> {
        if self.worktrees.is_empty() {
            if key.code == KeyCode::Char('r') && key.modifiers.contains(KeyModifiers::CONTROL) {
                self.refresh_worktrees().ok();
                return Ok(true);
            }
            return Ok(false);
        }

        if self.confirming_delete {
            match map_confirm_delete_key(key) {
                ConfirmDeleteAction::Confirm => {
                    self.delete_selected_worktree()?;
                    self.confirming_delete = false;
                    return Ok(true);
                }
                ConfirmDeleteAction::Cancel => {
                    self.confirming_delete = false;
                    return Ok(true);
                }
                ConfirmDeleteAction::None => return Ok(false),
            }
        }

        match map_worktree_key(key) {
            WorktreeAction::ToggleSidebar => {
                self.sidebar_open = !self.sidebar_open;
                Ok(true)
            }
            WorktreeAction::Refresh => {
                self.refresh_worktrees()?;
                Ok(true)
            }
            WorktreeAction::NavigateUp => {
                if self.selected_worktree > 0 {
                    self.selected_worktree -= 1;
                    self.prompt_expanded = false;
                }
                Ok(true)
            }
            WorktreeAction::NavigateDown => {
                if self.selected_worktree + 1 < self.worktrees.len() {
                    self.selected_worktree += 1;
                    self.prompt_expanded = false;
                }
                Ok(true)
            }
            WorktreeAction::JumpToFirst => {
                self.selected_worktree = 0;
                self.prompt_expanded = false;
                Ok(true)
            }
            WorktreeAction::JumpToLast => {
                if !self.worktrees.is_empty() {
                    self.selected_worktree = self.worktrees.len() - 1;
                    self.prompt_expanded = false;
                }
                Ok(true)
            }
            WorktreeAction::ToggleDetails => {
                self.show_details = !self.show_details;
                self.sidebar_open = self.show_details;
                Ok(true)
            }
            WorktreeAction::TogglePromptExpanded => {
                self.prompt_expanded = !self.prompt_expanded;
                Ok(true)
            }
            WorktreeAction::Open => {
                self.reopen_selected_worktree()?;
                Ok(true)
            }
            WorktreeAction::FocusOrOpen => {
                let focused = self.focus_selected_worktree()?;
                if !focused {
                    self.reopen_selected_worktree()?;
                }
                Ok(true)
            }
            WorktreeAction::InitiateDelete => {
                self.confirming_delete = true;
                Ok(true)
            }
            WorktreeAction::None => Ok(false),
        }
    }

    fn handle_launch_key(&mut self, key: KeyEvent) -> Result<bool> {
        match key.code {
            KeyCode::Tab if !key.modifiers.contains(KeyModifiers::SHIFT) => {
                self.launch_form.next_field();
                return Ok(true);
            }
            KeyCode::Tab if key.modifiers.contains(KeyModifiers::SHIFT) => {
                self.launch_form.prev_field();
                return Ok(true);
            }
            KeyCode::Up => {
                self.launch_form.prev_field();
                return Ok(true);
            }
            KeyCode::Down => {
                self.launch_form.next_field();
                return Ok(true);
            }
            KeyCode::Left => {
                if self.launch_form.focused == LaunchField::Preset {
                    self.launch_form.prev_preset();
                    return Ok(true);
                }
            }
            KeyCode::Right => {
                if self.launch_form.focused == LaunchField::Preset {
                    self.launch_form.next_preset();
                    return Ok(true);
                }
            }
            KeyCode::Enter => {
                if key.modifiers.contains(KeyModifiers::CONTROL)
                    || self.launch_form.focused == LaunchField::Launch
                {
                    self.submit_launch_form()?;
                    return Ok(true);
                }
                if self.launch_form.focused == LaunchField::Prompt {
                    self.launch_form.prompt.push('\n');
                } else {
                    self.launch_form.next_field();
                }
                return Ok(true);
            }
            KeyCode::Char(c) => match self.launch_form.focused {
                LaunchField::Repo => {
                    self.launch_form.repo.push(c);
                    return Ok(true);
                }
                LaunchField::Name => {
                    self.launch_form.name.push(c);
                    return Ok(true);
                }
                LaunchField::Prompt => {
                    self.launch_form.prompt.push(c);
                    return Ok(true);
                }
                _ => {}
            },
            KeyCode::Backspace => match self.launch_form.focused {
                LaunchField::Repo => {
                    self.launch_form.repo.pop();
                    return Ok(true);
                }
                LaunchField::Name => {
                    self.launch_form.name.pop();
                    return Ok(true);
                }
                LaunchField::Prompt => {
                    self.launch_form.prompt.pop();
                    return Ok(true);
                }
                _ => {}
            },
            _ => {}
        }

        Ok(false)
    }

    fn handle_settings_key(&mut self, key: KeyEvent) -> Result<bool> {
        match key.code {
            KeyCode::Up => {
                self.settings_form.prev_field();
                return Ok(true);
            }
            KeyCode::Down => {
                self.settings_form.next_field();
                return Ok(true);
            }
            KeyCode::Left => {
                self.adjust_setting(-1);
                return Ok(true);
            }
            KeyCode::Right => {
                self.adjust_setting(1);
                return Ok(true);
            }
            KeyCode::Enter => {
                self.save_settings()?;
                return Ok(true);
            }
            KeyCode::Backspace => {
                if self.settings_form.focused == SettingsField::DefaultPreset {
                    self.settings_form.app_settings.session.default_preset.pop();
                    return Ok(true);
                }
            }
            KeyCode::Char(c) => {
                if self.settings_form.focused == SettingsField::DefaultPreset {
                    self.settings_form
                        .app_settings
                        .session
                        .default_preset
                        .push(c);
                    return Ok(true);
                }
            }
            _ => {}
        }
        Ok(false)
    }

    fn handle_presets_key(&mut self, _key: KeyEvent) -> Result<bool> {
        Ok(false)
    }

    fn adjust_setting(&mut self, delta: i32) {
        use appsettings::TerminalType;
        match self.settings_form.focused {
            SettingsField::TerminalType => {
                // Toggle between the two options regardless of direction
                self.settings_form.app_settings.terminal.r#type =
                    match self.settings_form.app_settings.terminal.r#type {
                        TerminalType::ITerm2 => TerminalType::Terminal,
                        TerminalType::Terminal => TerminalType::ITerm2,
                    }
            }
            SettingsField::Maximize => {
                if delta != 0 {
                    self.settings_form.app_settings.terminal.maximize_on_launch =
                        !self.settings_form.app_settings.terminal.maximize_on_launch;
                }
            }
            SettingsField::DefaultPreset => {
                if let Some(cfg) = self.preset_config.as_ref() {
                    let mut names = cfg.presets.keys().cloned().collect::<Vec<_>>();
                    names.sort();
                    if !names.is_empty() {
                        let current = self
                            .settings_form
                            .app_settings
                            .session
                            .default_preset
                            .clone();
                        let idx = names.iter().position(|n| n == &current).unwrap_or(0);
                        let mut new_idx = idx as i32 + delta;
                        if new_idx < 0 {
                            new_idx = names.len() as i32 - 1;
                        }
                        if new_idx >= names.len() as i32 {
                            new_idx = 0;
                        }
                        self.settings_form.app_settings.session.default_preset =
                            names[new_idx as usize].clone();
                    }
                }
            }
            SettingsField::AutoClean => {
                if delta != 0 {
                    self.settings_form.app_settings.session.auto_clean_worktrees =
                        !self.settings_form.app_settings.session.auto_clean_worktrees;
                }
            }
            SettingsField::RetentionDays => {
                let mut days = self
                    .settings_form
                    .app_settings
                    .session
                    .worktree_retention_days;
                days = (days as i32 + delta).clamp(1, 365) as i64;
                self.settings_form
                    .app_settings
                    .session
                    .worktree_retention_days = days;
            }
        }
    }

    fn save_settings(&mut self) -> Result<()> {
        appsettings::save_app_settings(&self.data_dir, &self.settings_form.app_settings)?;
        self.app_settings = self.settings_form.app_settings.clone();
        self.set_status("Settings saved", false);
        Ok(())
    }

    fn submit_launch_form(&mut self) -> Result<()> {
        if let Some((repo, name, prompt, preset_name)) = self.launch_form.submit() {
            let preset = self
                .preset_config
                .as_ref()
                .and_then(|cfg| preset::get_preset(cfg, &preset_name))
                .unwrap_or_else(|| {
                    vec![preset::Worktree {
                        agent: "claude".to_string(),
                        n: 1,
                        commands: vec![],
                    }]
                });
            let opts = launcher::Options {
                repo: repo.clone(),
                name: name.clone(),
                prompt: prompt.clone(),
                preset_name: preset_name.clone(),
                multiplier: 1,
                data_dir: self.data_dir.clone(),
                preset,
                maximize_on_launch: self.app_settings.terminal.maximize_on_launch,
            };
            match launcher::launch(opts) {
                Ok(res) => {
                    self.set_status(
                        &format!(
                            "Launched {} session(s) in {} worktree(s)!",
                            res.sessions.len(),
                            res.terminal_window_count
                        ),
                        false,
                    );
                    self.refresh_worktrees().ok();
                }
                Err(err) => {
                    self.set_status(&format!("Launch failed: {}", err), true);
                }
            }
        } else {
            self.set_status("Fill repo, name, and prompt to launch", true);
        }
        Ok(())
    }

    fn focus_selected_worktree(&mut self) -> Result<bool> {
        if let Some(wt) = self.worktrees.get(self.selected_worktree) {
            if terminal::focus_worktree_window(&wt.path.to_string_lossy()).unwrap_or(false) {
                self.set_status(&format!("Focused {}", wt.name), false);
                return Ok(true);
            }
            if terminal::focus_worktree_window_by_branch(&wt.branch).unwrap_or(false) {
                self.set_status(&format!("Focused {}", wt.name), false);
                return Ok(true);
            }
            self.set_status("No existing iTerm2 window for this worktree", true);
        }
        Ok(false)
    }

    fn reopen_selected_worktree(&mut self) -> Result<()> {
        if let Some(wt) = self.worktrees.get(self.selected_worktree) {
            // Build sessions for existing worktree
            let preset = self
                .preset_config
                .as_ref()
                .and_then(|cfg| preset::get_preset(cfg, &wt.preset_name))
                .unwrap_or_else(|| {
                    if wt.agents.is_empty() {
                        vec![preset::Worktree {
                            agent: "claude".to_string(),
                            n: 1,
                            commands: vec![],
                        }]
                    } else {
                        vec![preset::Worktree {
                            agent: wt.agents[0].clone(),
                            n: 1,
                            commands: vec![],
                        }]
                    }
                });

            let mut sessions = Vec::new();
            let branch = wt.branch.clone();
            let path_str = wt.path.to_string_lossy().to_string();
            let agents = if wt.agents.is_empty() {
                vec![wt.display_agent()]
            } else {
                wt.agents.clone()
            };
            let activity_path = self
                .data_dir
                .join("activity")
                .join(format!("{}.log", branch));
            for agent in agents {
                let mut s = terminal::SessionInfo::agent_session(&path_str, &branch, &agent);
                s.activity_log = Some(activity_path.to_string_lossy().to_string());
                sessions.push(s);
            }
            for w in &preset {
                for cmd in &w.commands {
                    let color = preset::parse_hex_color(&cmd.color);
                    sessions.push(terminal::SessionInfo::custom_command(
                        &cmd.command,
                        &cmd.display_title(),
                        color,
                        &path_str,
                        &branch,
                    ));
                }
            }

            if sessions.is_empty() {
                self.set_status("No sessions to launch for this worktree", true);
                return Ok(());
            }

            // Use saved prompt when reopening; fall back to a simple continuation prompt.
            let prompt = if !wt.prompt.is_empty() {
                wt.prompt.clone()
            } else {
                format!("Continue working on {}", wt.branch)
            };

            let mgr = terminal::TerminalManager::new(self.app_settings.terminal.maximize_on_launch);
            match mgr.launch_sessions(&sessions, &prompt) {
                Ok(_count) => {
                    self.set_status(
                        &format!("Opened {} session(s) for {}", sessions.len(), wt.name),
                        false,
                    );
                    self.refresh_worktrees().ok();
                }
                Err(err) => {
                    self.set_status(&format!("Launch failed: {}", err), true);
                }
            }
        }
        Ok(())
    }

    fn delete_selected_worktree(&mut self) -> Result<()> {
        if let Some(wt) = self.worktrees.get(self.selected_worktree) {
            fs::remove_dir_all(&wt.path)?;
            self.set_status(&format!("Deleted {}", wt.name), false);
            self.refresh_worktrees()?;
        }
        Ok(())
    }

    fn set_status(&mut self, msg: &str, is_error: bool) {
        self.status = Some((msg.to_string(), is_error));
    }

    fn next_view(&mut self) {
        self.view = match self.view {
            View::Worktrees => View::Launch,
            View::Launch => View::Settings,
            View::Settings => View::Presets,
            View::Presets => View::Worktrees,
        }
    }

    fn prev_view(&mut self) {
        self.view = match self.view {
            View::Worktrees => View::Presets,
            View::Launch => View::Worktrees,
            View::Settings => View::Launch,
            View::Presets => View::Settings,
        }
    }

    fn draw(&mut self, f: &mut Frame) {
        let size = f.area();
        let mut constraints = vec![Constraint::Length(3), Constraint::Min(0)];
        if self.app_settings.ui.show_status_bar {
            constraints.push(Constraint::Length(1));
        }
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints(constraints)
            .split(size);

        self.draw_header(f, chunks[0]);

        // Fill the body with the app background so margins match the worktrees page
        if let Some(body_area) = chunks.get(1) {
            let background = Block::default().style(Style::default().bg(SURFACE_BG));
            f.render_widget(background, *body_area);
        }

        if self.sidebar_open {
            let body_chunks = Layout::default()
                .direction(Direction::Horizontal)
                .constraints([Constraint::Percentage(70), Constraint::Percentage(30)].as_ref())
                .split(chunks[1]);
            self.draw_body(f, body_chunks[0]);
            self.draw_sidebar(f, body_chunks[1]);
        } else {
            self.draw_body(f, chunks[1]);
        }

        if self.app_settings.ui.show_status_bar {
            if let Some(status_area) = chunks.get(2) {
                self.draw_status(f, *status_area);
            }
        }

        if self.confirming_delete {
            self.draw_confirm_dialog(f);
        }

        if self.help_expanded {
            self.draw_help_overlay(f);
        }
    }

    fn draw_header(&self, f: &mut Frame, area: Rect) {
        let tabs = [
            ("Worktrees", View::Worktrees),
            ("Launch", View::Launch),
            ("Settings", View::Settings),
            ("Presets", View::Presets),
        ];

        let mut spans: Vec<Span> = Vec::new();
        for (idx, (label, view)) in tabs.iter().enumerate() {
            let active = self.view == *view;
            if idx > 0 {
                spans.push(Span::styled("  ", Style::default()));
            }
            if active {
                spans.push(Span::styled(
                    format!(" {} ", label),
                    Style::default()
                        .fg(Color::Rgb(20, 22, 28))
                        .bg(ACCENT)
                        .add_modifier(Modifier::BOLD),
                ));
            } else {
                spans.push(Span::styled(
                    format!(" {} ", label),
                    Style::default().fg(DIM_TEXT).add_modifier(Modifier::DIM),
                ));
            }
        }

        let line = Line::from(spans);
        let nav_hint = Line::from(vec![
            Span::styled(
                "Tab",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" navigate  ", Style::default().fg(DIM_TEXT)),
            Span::styled(
                "?",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" help  ", Style::default().fg(DIM_TEXT)),
            Span::styled(
                "Ctrl+C",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" quit", Style::default().fg(DIM_TEXT)),
        ]);

        let block = Block::default()
            .borders(Borders::BOTTOM)
            .border_type(BorderType::Plain)
            .border_style(Style::default().fg(LIGHT_BORDER))
            .style(Style::default().bg(HEADER_BG));
        let inner = block.inner(area);
        f.render_widget(block, area);

        let rows = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Length(1), Constraint::Length(1)].as_ref())
            .split(inner);

        let cols = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Percentage(65), Constraint::Percentage(35)].as_ref())
            .split(rows[0]);

        let tabs_para = Paragraph::new(line).style(Style::default().bg(HEADER_BG));
        f.render_widget(tabs_para, cols[0]);

        let logo = Paragraph::new(Line::from(vec![
            Span::styled(
                "Dispatch",
                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" by ", Style::default().fg(DIM_TEXT)),
            Span::styled(
                "Groq",
                Style::default()
                    .fg(Color::White)
                    .add_modifier(Modifier::BOLD),
            ),
        ]))
        .alignment(Alignment::Right)
        .style(Style::default().bg(HEADER_BG));
        f.render_widget(logo, cols[1]);

        let nav = Paragraph::new(nav_hint)
            .alignment(Alignment::Center)
            .style(Style::default().bg(HEADER_BG));
        f.render_widget(nav, rows[1]);
    }

    fn draw_body(&self, f: &mut Frame, area: Rect) {
        match self.view {
            View::Worktrees => self.draw_worktrees(f, area),
            View::Launch => self.draw_launch(f, area),
            View::Settings => self.draw_settings(f, area),
            View::Presets => self.draw_presets(f, area),
        }
    }

    fn draw_status(&self, f: &mut Frame, area: Rect) {
        let content = if let Some((msg, is_error)) = &self.status {
            let color = if *is_error {
                ERROR_COLOR
            } else {
                SUCCESS_COLOR
            };
            Line::from(Span::styled(msg.clone(), Style::default().fg(color)))
        } else {
            Line::from(vec![
                Span::styled("Ready  ", Style::default().fg(DIM_TEXT)),
                Span::styled("|  ", Style::default().fg(LIGHT_BORDER)),
                Span::styled("Ctrl+C", Style::default().fg(ACCENT_DIM)),
                Span::styled(" quit  ", Style::default().fg(DIM_TEXT)),
                Span::styled("Tab", Style::default().fg(ACCENT_DIM)),
                Span::styled(" switch views", Style::default().fg(DIM_TEXT)),
            ])
        };
        let block = Block::default()
            .borders(Borders::TOP)
            .border_style(Style::default().fg(LIGHT_BORDER))
            .style(Style::default().bg(HEADER_BG));
        let text = Paragraph::new(content).block(block);
        f.render_widget(text, area);
    }

    fn draw_worktrees(&self, f: &mut Frame, area: Rect) {
        if self.worktrees_loading {
            let loading_block = Block::default()
                .borders(Borders::ALL)
                .border_type(BorderType::Rounded)
                .border_style(Style::default().fg(LIGHT_BORDER))
                .style(Style::default().bg(SURFACE_BG));
            let text = Paragraph::new(Line::from(vec![
                Span::styled("◐ ", Style::default().fg(ACCENT)),
                Span::styled("Loading worktrees...", Style::default().fg(DIM_TEXT)),
            ]))
            .block(loading_block)
            .alignment(Alignment::Center);
            f.render_widget(text, area);
            return;
        }
        if self.worktrees.is_empty() {
            let empty_block = Block::default()
                .borders(Borders::ALL)
                .border_type(BorderType::Rounded)
                .border_style(Style::default().fg(LIGHT_BORDER))
                .style(Style::default().bg(SURFACE_BG))
                .title(Span::styled(
                    " Worktrees ",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ));
            let msg = Paragraph::new(vec![
                Line::from(""),
                Line::from(Span::styled(
                    "No worktrees found",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                )),
                Line::from(""),
                Line::from(vec![
                    Span::styled("Use the ", Style::default().fg(DIM_TEXT)),
                    Span::styled(
                        "Launch",
                        Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                    ),
                    Span::styled(" view or ", Style::default().fg(DIM_TEXT)),
                    Span::styled(
                        "CLI",
                        Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                    ),
                    Span::styled(" to create sessions", Style::default().fg(DIM_TEXT)),
                ]),
            ])
            .block(empty_block)
            .alignment(Alignment::Center);
            f.render_widget(msg, area);
            return;
        }

        let selected = self.worktrees.get(self.selected_worktree);

        // Optional actions bar at bottom
        let main_sections = if self.actions_bar {
            Layout::default()
                .direction(Direction::Vertical)
                .constraints([Constraint::Min(0), Constraint::Length(1)].as_ref())
                .split(area)
        } else {
            Layout::default()
                .direction(Direction::Vertical)
                .constraints([Constraint::Min(0)].as_ref())
                .split(area)
        };

        // Two columns: left (combined summary) and right (full-height activity)
        let columns = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Percentage(65), Constraint::Percentage(35)].as_ref())
            .split(main_sections[0]);

        let left_sections = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Percentage(45), Constraint::Min(0)].as_ref())
            .split(columns[0]);

        // Combined summary block with repo/branch/agent plus commits and files
        let mut summary_lines: Vec<Line> = Vec::new();
        let repo_text = selected
            .and_then(|w| {
                if w.repo.is_empty() {
                    None
                } else {
                    Some(w.repo.clone())
                }
            })
            .unwrap_or_else(|| "—".to_string());
        let branch_text = selected
            .and_then(|w| {
                if w.branch.is_empty() {
                    None
                } else {
                    Some(w.branch.clone())
                }
            })
            .unwrap_or_else(|| "—".to_string());
        let agent_text = selected
            .map(|w| {
                if w.agents.is_empty() {
                    "—".to_string()
                } else {
                    w.agents.join(", ")
                }
            })
            .unwrap_or_else(|| "—".to_string());

        let mut prompt_preview = selected
            .map(|w| w.prompt.clone())
            .unwrap_or_else(|| "—".to_string());
        if prompt_preview.is_empty() {
            prompt_preview = "—".to_string();
        } else if !self.prompt_expanded && prompt_preview.len() > 140 {
            prompt_preview.truncate(140);
            prompt_preview.push_str("… ");
        }

        let label_style = Style::default().fg(DIM_TEXT);
        let value_style = Style::default().fg(Color::White);

        summary_lines.push(Line::from(vec![
            Span::styled("Repo    ", label_style),
            Span::styled(repo_text, value_style),
        ]));
        summary_lines.push(Line::from(vec![
            Span::styled("Branch  ", label_style),
            Span::styled(branch_text, value_style),
        ]));
        summary_lines.push(Line::from(vec![
            Span::styled("Agent   ", label_style),
            Span::styled(agent_text, value_style),
        ]));
        summary_lines.push(Line::from(vec![
            Span::styled("Prompt  ", label_style),
            Span::styled(
                prompt_preview.clone(),
                Style::default().fg(Color::Rgb(180, 180, 190)),
            ),
        ]));
        if !self.prompt_expanded && selected.map(|w| w.prompt.len() > 140).unwrap_or(false) {
            summary_lines.push(Line::from(vec![
                Span::styled("         ", Style::default()),
                Span::styled("press ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "e",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" to expand", Style::default().fg(DIM_TEXT)),
            ]));
        }
        summary_lines.push(Line::from("")); // spacer

        let section_style = Style::default().fg(ACCENT).add_modifier(Modifier::BOLD);

        summary_lines.push(Line::from(Span::styled("Recent Commits", section_style)));
        if let Some(wt) = selected {
            if wt.recent_commits.is_empty() {
                summary_lines.push(Line::from(Span::styled(
                    "   No recent commits",
                    Style::default().fg(DIM_TEXT),
                )));
            } else {
                for c in wt.recent_commits.iter().take(6) {
                    summary_lines.push(Line::from(vec![
                        Span::styled("   ", Style::default()),
                        Span::styled(c.clone(), Style::default().fg(Color::Rgb(180, 180, 190))),
                    ]));
                }
            }
        } else {
            summary_lines.push(Line::from(Span::styled(
                "   No selection",
                Style::default().fg(DIM_TEXT),
            )));
        }
        summary_lines.push(Line::from("")); // spacer

        summary_lines.push(Line::from(Span::styled("Files Changed", section_style)));
        if let Some(wt) = selected {
            if wt.file_stats.is_empty() {
                summary_lines.push(Line::from(Span::styled(
                    "   No uncommitted changes",
                    Style::default().fg(DIM_TEXT),
                )));
            } else {
                for fs in wt.file_stats.iter().take(10) {
                    summary_lines.push(Line::from(vec![
                        Span::styled("   ", Style::default()),
                        Span::styled(
                            format!("+{:<3}", fs.adds),
                            Style::default().fg(SUCCESS_COLOR),
                        ),
                        Span::styled(
                            format!("-{:<3}", fs.deletes),
                            Style::default().fg(ERROR_COLOR),
                        ),
                        Span::styled(
                            format!(" {}", fs.path),
                            Style::default().fg(Color::Rgb(180, 180, 190)),
                        ),
                    ]));
                }
            }
        } else {
            summary_lines.push(Line::from(Span::styled(
                "   No selection",
                Style::default().fg(DIM_TEXT),
            )));
        }

        let summary = Paragraph::new(summary_lines)
            .block(
                Block::default()
                    .title(Span::styled(
                        " Current Session ",
                        Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                    ))
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded)
                    .border_style(Style::default().fg(LIGHT_BORDER))
                    .style(Style::default().bg(SURFACE_BG)),
            )
            .wrap(Wrap { trim: false });
        f.render_widget(summary, left_sections[0]);

        // Activity stream uses the full right column height
        let mut activity_lines: Vec<Line> = vec![];
        if let Some(wt) = selected {
            let (status_label, status_fg, status_bg) = if wt.activity_active {
                ("Active", STATUS_ACTIVE_FG, STATUS_ACTIVE_BG)
            } else {
                ("Idle", STATUS_IDLE_FG, STATUS_IDLE_BG)
            };
            activity_lines.push(Line::from(vec![
                Span::styled("Status  ", label_style),
                Span::styled(
                    format!(" {} ", status_label),
                    Style::default()
                        .fg(status_fg)
                        .bg(status_bg)
                        .add_modifier(Modifier::BOLD),
                ),
            ]));
            if let Some(agent) = wt.agents.first() {
                activity_lines.push(Line::from(vec![
                    Span::styled("Agent   ", label_style),
                    Span::styled(agent.to_string(), value_style),
                ]));
            }
            if let Some(p) = &wt.activity_log {
                activity_lines.push(Line::from(vec![
                    Span::styled("Log     ", label_style),
                    Span::styled(util::display_path(p), Style::default().fg(DIM_TEXT)),
                ]));
            }
            activity_lines.push(Line::from("")); // spacer
            activity_lines.push(Line::from(Span::styled("Recent Output", section_style)));
            activity_lines.push(Line::from(Span::styled(
                "─".repeat(30),
                Style::default().fg(LIGHT_BORDER),
            )));
            if wt.activity_recent.is_empty() {
                activity_lines.push(Line::from(Span::styled(
                    "No recent output",
                    Style::default().fg(DIM_TEXT),
                )));
            } else {
                for line in &wt.activity_recent {
                    let lower = line.to_lowercase();
                    let line_style = if lower.contains("error") || lower.contains("fail") {
                        Style::default().fg(ERROR_COLOR)
                    } else if lower.contains("success")
                        || lower.contains("done")
                        || lower.contains("complete")
                    {
                        Style::default().fg(SUCCESS_COLOR)
                    } else if lower.contains("warn") {
                        Style::default().fg(WARNING_COLOR)
                    } else {
                        Style::default().fg(Color::Rgb(170, 175, 185))
                    };
                    activity_lines.push(Line::from(vec![Span::styled(line.clone(), line_style)]));
                }
            }
            let activity = Paragraph::new(activity_lines)
                .block(
                    Block::default()
                        .title(Span::styled(
                            " Agent Activity ",
                            Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                        ))
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded)
                        .border_style(Style::default().fg(LIGHT_BORDER))
                        .style(Style::default().bg(SURFACE_BG)),
                )
                .wrap(Wrap { trim: true });
            f.render_widget(activity, columns[1]);
        } else {
            let activity =
                Paragraph::new(Span::styled("No selection", Style::default().fg(DIM_TEXT)))
                    .block(
                        Block::default()
                            .title(Span::styled(
                                " Agent Activity ",
                                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                            ))
                            .borders(Borders::ALL)
                            .border_type(BorderType::Rounded)
                            .border_style(Style::default().fg(LIGHT_BORDER))
                            .style(Style::default().bg(SURFACE_BG)),
                    )
                    .wrap(Wrap { trim: false });
            f.render_widget(activity, columns[1]);
        }

        let rows: Vec<Row> = self
            .worktrees
            .iter()
            .enumerate()
            .map(|(idx, wt)| {
                let (status_label, status_fg, status_bg) = if wt.activity_active {
                    ("Active", STATUS_ACTIVE_FG, STATUS_ACTIVE_BG)
                } else {
                    ("Idle", STATUS_IDLE_FG, STATUS_IDLE_BG)
                };
                let zebra = if idx % 2 == 0 { Some(ZEBRA_BG) } else { None };
                let mut style = if idx == self.selected_worktree {
                    Style::default()
                        .bg(ACCENT_HIGHLIGHT_BG)
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD)
                } else {
                    Style::default().fg(Color::Rgb(200, 205, 215))
                };
                if idx != self.selected_worktree {
                    if let Some(bg) = zebra {
                        style = style.bg(bg);
                    }
                }

                // Truncate name for cleaner look
                let name_display = if wt.name.len() > 20 {
                    format!("{}…", &wt.name[..19])
                } else {
                    wt.name.clone()
                };

                Row::new(vec![
                    Cell::from(name_display),
                    Cell::from(wt.repo.clone()),
                    Cell::from(wt.branch.clone()),
                    Cell::from(Line::from(vec![
                        Span::styled(format!("+{}", wt.adds), Style::default().fg(SUCCESS_COLOR)),
                        Span::styled(" ", Style::default().fg(DIM_TEXT)),
                        Span::styled(format!("-{}", wt.deletes), Style::default().fg(ERROR_COLOR)),
                    ])),
                    Cell::from(wt.display_agent()),
                    Cell::from(Span::styled(
                        format!(" {} ", status_label),
                        Style::default().fg(status_fg).bg(status_bg),
                    )),
                    Cell::from(wt.last_commit.clone()),
                ])
                .style(style)
            })
            .collect();

        let widths = [
            Constraint::Percentage(16),
            Constraint::Percentage(18),
            Constraint::Percentage(16),
            Constraint::Percentage(10),
            Constraint::Percentage(10),
            Constraint::Percentage(10),
            Constraint::Percentage(20),
        ];

        let header = Row::new(vec![
            Cell::from(Span::styled("Name", Style::default().fg(DIM_TEXT))),
            Cell::from(Span::styled("Repo", Style::default().fg(DIM_TEXT))),
            Cell::from(Span::styled("Branch", Style::default().fg(DIM_TEXT))),
            Cell::from(Span::styled("Changes", Style::default().fg(DIM_TEXT))),
            Cell::from(Span::styled("Agent", Style::default().fg(DIM_TEXT))),
            Cell::from(Span::styled("Status", Style::default().fg(DIM_TEXT))),
            Cell::from(Span::styled("Last Commit", Style::default().fg(DIM_TEXT))),
        ])
        .style(Style::default().bg(HEADER_BG))
        .height(1);

        let table = Table::new(rows, widths)
            .header(header)
            .column_spacing(1)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded)
                    .border_style(Style::default().fg(LIGHT_BORDER))
                    .title(Span::styled(
                        " Worktrees ",
                        Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                    ))
                    .style(Style::default().bg(SURFACE_BG)),
            );
        f.render_widget(table, left_sections[1]);

        if self.actions_bar {
            let actions_text = Paragraph::new(Line::from(vec![
                Span::styled(
                    "Enter",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" focus/open  ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "x",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" delete  ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "Ctrl+R",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" refresh  ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "d",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" details", Style::default().fg(DIM_TEXT)),
            ]))
            .alignment(Alignment::Center)
            .style(Style::default().bg(HEADER_BG));
            f.render_widget(actions_text, main_sections[1]);
        }
    }

    fn draw_sidebar(&self, f: &mut Frame, area: Rect) {
        if !self.show_details {
            let text = Paragraph::new(Line::from(vec![
                Span::styled("Press ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "d",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" to toggle details", Style::default().fg(DIM_TEXT)),
            ]))
            .wrap(Wrap { trim: true })
            .alignment(Alignment::Center)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded)
                    .border_style(Style::default().fg(LIGHT_BORDER))
                    .style(Style::default().bg(SURFACE_BG))
                    .title(Span::styled(" Details ", Style::default().fg(DIM_TEXT))),
            );
            f.render_widget(text, area);
            return;
        }
        if let Some(wt) = self.worktrees.get(self.selected_worktree) {
            let mut lines: Vec<Line> = Vec::new();
            let label_style = Style::default().fg(DIM_TEXT);
            let value_style = Style::default().fg(Color::White);

            lines.push(Line::from(vec![Span::styled(
                wt.name.clone(),
                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
            )]));
            lines.push(Line::from(Span::styled(
                "─".repeat(25),
                Style::default().fg(LIGHT_BORDER),
            )));
            lines.push(Line::from(vec![
                Span::styled("Path    ", label_style),
                Span::styled(util::display_path(&wt.path), value_style),
            ]));
            lines.push(Line::from(vec![
                Span::styled("Repo    ", label_style),
                Span::styled(wt.repo.clone(), value_style),
            ]));
            lines.push(Line::from(vec![
                Span::styled("Branch  ", label_style),
                Span::styled(wt.branch.clone(), value_style),
            ]));
            if wt.has_meta {
                lines.push(Line::from(vec![
                    Span::styled("Preset  ", label_style),
                    Span::styled(
                        if wt.preset_name.is_empty() {
                            "—"
                        } else {
                            &wt.preset_name
                        },
                        value_style,
                    ),
                ]));
            }
            if !wt.agents.is_empty() {
                lines.push(Line::from(vec![
                    Span::styled("Agents  ", label_style),
                    Span::styled(wt.agents.join(", "), value_style),
                ]));
            }
            if let Some(created) = wt.created_at {
                lines.push(Line::from(vec![
                    Span::styled("Created ", label_style),
                    Span::styled(created.format("%Y-%m-%d %H:%M").to_string(), value_style),
                ]));
            }
            lines.push(Line::from(vec![
                Span::styled("Commit  ", label_style),
                Span::styled(wt.last_commit.clone(), value_style),
            ]));

            if !wt.prompt.is_empty() {
                lines.push(Line::from(""));
                lines.push(Line::from(Span::styled(
                    "Prompt",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                )));
                lines.push(Line::from(Span::styled(
                    wt.prompt.clone(),
                    Style::default().fg(Color::Rgb(180, 180, 190)),
                )));
            }
            if !wt.file_stats.is_empty() {
                lines.push(Line::from(""));
                lines.push(Line::from(Span::styled(
                    "Files Changed",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                )));
                for fs in wt.file_stats.iter().take(8) {
                    lines.push(Line::from(vec![
                        Span::styled(
                            format!("+{:<3}", fs.adds),
                            Style::default().fg(SUCCESS_COLOR),
                        ),
                        Span::styled(
                            format!("-{:<3}", fs.deletes),
                            Style::default().fg(ERROR_COLOR),
                        ),
                        Span::styled(
                            format!(" {}", fs.path),
                            Style::default().fg(Color::Rgb(180, 180, 190)),
                        ),
                    ]));
                }
            }
            let text = Paragraph::new(lines)
                .block(
                    Block::default()
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded)
                        .border_style(Style::default().fg(LIGHT_BORDER))
                        .style(Style::default().bg(SURFACE_BG))
                        .title(Span::styled(
                            " Details ",
                            Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                        )),
                )
                .wrap(Wrap { trim: false });
            f.render_widget(text, area);
        }
    }

    fn draw_launch(&self, f: &mut Frame, area: Rect) {
        // Create a centered form layout
        let outer = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([
                Constraint::Percentage(15),
                Constraint::Percentage(70),
                Constraint::Percentage(15),
            ])
            .split(area);

        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints(
                [
                    Constraint::Length(1), // spacing
                    Constraint::Length(3),
                    Constraint::Length(3),
                    Constraint::Length(7),
                    Constraint::Length(3),
                    Constraint::Length(3),
                    Constraint::Min(0),
                ]
                .as_ref(),
            )
            .split(outer[1]);

        // Form title
        let title = Paragraph::new(Span::styled(
            "Launch New Session",
            Style::default()
                .fg(Color::White)
                .add_modifier(Modifier::BOLD),
        ))
        .alignment(Alignment::Center);
        f.render_widget(title, chunks[0]);

        let repo = Paragraph::new(if self.launch_form.repo.is_empty() {
            Span::styled("owner/repo", Style::default().fg(DIM_TEXT))
        } else {
            Span::styled(
                self.launch_form.repo.as_str(),
                Style::default().fg(Color::White),
            )
        })
        .block(self.input_block(
            "Repository",
            self.launch_form.focused == LaunchField::Repo,
            "",
        ))
        .wrap(Wrap { trim: true });
        f.render_widget(repo, chunks[1]);

        let name = Paragraph::new(if self.launch_form.name.is_empty() {
            Span::styled("feature-name", Style::default().fg(DIM_TEXT))
        } else {
            Span::styled(
                self.launch_form.name.as_str(),
                Style::default().fg(Color::White),
            )
        })
        .block(self.input_block(
            "Branch prefix",
            self.launch_form.focused == LaunchField::Name,
            "",
        ))
        .wrap(Wrap { trim: true });
        f.render_widget(name, chunks[2]);

        let prompt = Paragraph::new(if self.launch_form.prompt.is_empty() {
            Span::styled("What should the agent do?", Style::default().fg(DIM_TEXT))
        } else {
            Span::styled(
                self.launch_form.prompt.as_str(),
                Style::default().fg(Color::White),
            )
        })
        .block(self.input_block(
            "Prompt",
            self.launch_form.focused == LaunchField::Prompt,
            "",
        ))
        .wrap(Wrap { trim: true });
        f.render_widget(prompt, chunks[3]);

        let preset = {
            let text = if self.launch_form.preset_names.is_empty() {
                Line::from(Span::styled("default", Style::default().fg(Color::White)))
            } else {
                let count = self.launch_form.preset_names.len();
                Line::from(vec![
                    Span::styled("< ", Style::default().fg(ACCENT_DIM)),
                    Span::styled(
                        self.launch_form.preset(),
                        Style::default()
                            .fg(Color::White)
                            .add_modifier(Modifier::BOLD),
                    ),
                    Span::styled(" >", Style::default().fg(ACCENT_DIM)),
                    Span::styled(
                        format!("   {}/{}", self.launch_form.preset_idx + 1, count),
                        Style::default().fg(DIM_TEXT),
                    ),
                ])
            };
            Paragraph::new(text)
                .block(self.input_block(
                    "Preset",
                    self.launch_form.focused == LaunchField::Preset,
                    "",
                ))
                .wrap(Wrap { trim: true })
        };
        f.render_widget(preset, chunks[4]);

        let btn_focused = self.launch_form.focused == LaunchField::Launch;
        let btn_block = Block::default()
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(if btn_focused {
                Style::default().fg(ACCENT).bg(SURFACE_BG)
            } else {
                Style::default().fg(LIGHT_BORDER).bg(SURFACE_BG)
            })
            .style(Style::default().bg(SURFACE_BG));
        let btn = Paragraph::new(Span::styled(
            "Launch Session",
            Style::default()
                .fg(if btn_focused { ACCENT } else { Color::White })
                .add_modifier(if btn_focused {
                    Modifier::BOLD
                } else {
                    Modifier::empty()
                }),
        ))
        .block(btn_block)
        .alignment(Alignment::Center);
        f.render_widget(btn, chunks[5]);

        let help = Paragraph::new(Line::from(vec![
            Span::styled(
                "Tab",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" move  ", Style::default().fg(DIM_TEXT)),
            Span::styled(
                "←/→",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" preset  ", Style::default().fg(DIM_TEXT)),
            Span::styled(
                "Ctrl+Enter",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" launch", Style::default().fg(DIM_TEXT)),
        ]))
        .alignment(Alignment::Center);
        f.render_widget(help, chunks[6]);
    }

    fn input_block(&self, label: &str, focused: bool, _hint: &str) -> Block<'static> {
        Block::default()
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(if focused {
                Style::default().fg(ACCENT).bg(SURFACE_BG)
            } else {
                Style::default().fg(LIGHT_BORDER).bg(SURFACE_BG)
            })
            .style(Style::default().bg(SURFACE_BG))
            .title(Span::styled(
                format!(" {} ", label),
                Style::default().fg(if focused { ACCENT } else { DIM_TEXT }),
            ))
    }

    fn draw_settings(&self, f: &mut Frame, area: Rect) {
        // Center the settings panel
        let outer = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([
                Constraint::Percentage(20),
                Constraint::Percentage(60),
                Constraint::Percentage(20),
            ])
            .split(area);

        let items: Vec<(String, String, bool)> = vec![
            (
                "Terminal Type".to_string(),
                format!("< {:?} >", self.settings_form.app_settings.terminal.r#type),
                self.settings_form.focused == SettingsField::TerminalType,
            ),
            (
                "Maximize on Launch".to_string(),
                if self.settings_form.app_settings.terminal.maximize_on_launch {
                    "Enabled".to_string()
                } else {
                    "Disabled".to_string()
                },
                self.settings_form.focused == SettingsField::Maximize,
            ),
            (
                "Default Preset".to_string(),
                format!(
                    "< {} >",
                    self.settings_form.app_settings.session.default_preset
                ),
                self.settings_form.focused == SettingsField::DefaultPreset,
            ),
            (
                "Auto Clean Worktrees".to_string(),
                if self.settings_form.app_settings.session.auto_clean_worktrees {
                    "Enabled".to_string()
                } else {
                    "Disabled".to_string()
                },
                self.settings_form.focused == SettingsField::AutoClean,
            ),
            (
                "Retention Days".to_string(),
                format!(
                    "< {} days >",
                    self.settings_form
                        .app_settings
                        .session
                        .worktree_retention_days
                ),
                self.settings_form.focused == SettingsField::RetentionDays,
            ),
        ];

        let rows: Vec<Row> = items
            .into_iter()
            .enumerate()
            .map(|(idx, (label, value, focused))| {
                let bg = if focused {
                    ACCENT_HIGHLIGHT_BG
                } else if idx % 2 == 0 {
                    ZEBRA_BG
                } else {
                    SURFACE_BG
                };
                let style = if focused {
                    Style::default()
                        .fg(ACCENT)
                        .bg(bg)
                        .add_modifier(Modifier::BOLD)
                } else {
                    Style::default().fg(Color::Rgb(200, 205, 215)).bg(bg)
                };
                Row::new(vec![
                    Cell::from(Span::styled(
                        label.to_string(),
                        if focused {
                            Style::default().fg(ACCENT)
                        } else {
                            Style::default().fg(DIM_TEXT)
                        },
                    )),
                    Cell::from(Span::styled(
                        value,
                        if focused {
                            Style::default()
                                .fg(Color::White)
                                .add_modifier(Modifier::BOLD)
                        } else {
                            Style::default().fg(Color::Rgb(180, 180, 190))
                        },
                    )),
                ])
                .style(style)
            })
            .collect();

        let header = Row::new(vec![
            Cell::from(Span::styled("Setting", Style::default().fg(DIM_TEXT))),
            Cell::from(Span::styled("Value", Style::default().fg(DIM_TEXT))),
        ])
        .style(Style::default().bg(HEADER_BG))
        .height(1);

        let table = Table::new(
            rows,
            [Constraint::Percentage(50), Constraint::Percentage(50)],
        )
        .header(header)
        .block(
            Block::default()
                .borders(Borders::ALL)
                .border_type(BorderType::Rounded)
                .border_style(Style::default().fg(LIGHT_BORDER))
                .style(Style::default().bg(SURFACE_BG))
                .title(Span::styled(
                    " Settings ",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                )),
        );
        f.render_widget(table, outer[1]);

        // Help text at bottom
        let help_area = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Min(0), Constraint::Length(1)])
            .split(outer[1]);

        let help = Paragraph::new(Line::from(vec![
            Span::styled(
                "↑/↓",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" select  ", Style::default().fg(DIM_TEXT)),
            Span::styled(
                "←/→",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" change  ", Style::default().fg(DIM_TEXT)),
            Span::styled(
                "Enter",
                Style::default().fg(ACCENT_DIM).add_modifier(Modifier::BOLD),
            ),
            Span::styled(" save", Style::default().fg(DIM_TEXT)),
        ]))
        .alignment(Alignment::Center);
        f.render_widget(help, help_area[1]);
    }

    fn draw_presets(&self, f: &mut Frame, area: Rect) {
        // Center the presets panel
        let outer = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([
                Constraint::Percentage(20),
                Constraint::Percentage(60),
                Constraint::Percentage(20),
            ])
            .split(area);

        if let Some(cfg) = &self.preset_config {
            let mut items = Vec::new();
            for (name, preset) in cfg.presets.iter() {
                let mut lines: Vec<Line> = Vec::new();
                lines.push(Line::from(Span::styled(
                    name.clone(),
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                )));
                lines.push(Line::from(Span::styled(
                    "─".repeat(40),
                    Style::default().fg(LIGHT_BORDER),
                )));
                for (idx, wt) in preset.iter().enumerate() {
                    lines.push(Line::from(vec![
                        Span::styled(format!("  {}. ", idx + 1), Style::default().fg(DIM_TEXT)),
                        Span::styled("Agent: ", Style::default().fg(DIM_TEXT)),
                        Span::styled(
                            wt.agent.clone(),
                            Style::default().fg(Color::Rgb(180, 180, 190)),
                        ),
                    ]));
                    for cmd in &wt.commands {
                        let title = cmd.display_title();
                        lines.push(Line::from(vec![
                            Span::styled("       -> ", Style::default().fg(DIM_TEXT)),
                            Span::styled(title, Style::default().fg(Color::Rgb(150, 150, 160))),
                        ]));
                    }
                }
                lines.push(Line::from("")); // spacer between presets
                items.push(ListItem::new(lines));
            }
            let list = List::new(items)
                .block(
                    Block::default()
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded)
                        .border_style(Style::default().fg(LIGHT_BORDER))
                        .style(Style::default().bg(SURFACE_BG))
                        .title(Span::styled(
                            " Presets ",
                            Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                        )),
                )
                .highlight_style(Style::default().fg(ACCENT));
            f.render_widget(list, outer[1]);
        } else {
            let empty_msg = Paragraph::new(vec![
                Line::from(""),
                Line::from(Span::styled(
                    "No presets configured",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                )),
                Line::from(""),
                Line::from(Span::styled(
                    "Create a settings.yaml file in your data directory",
                    Style::default().fg(DIM_TEXT),
                )),
                Line::from(Span::styled(
                    "to define custom agent presets.",
                    Style::default().fg(DIM_TEXT),
                )),
            ])
            .alignment(Alignment::Center)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded)
                    .border_style(Style::default().fg(LIGHT_BORDER))
                    .style(Style::default().bg(SURFACE_BG))
                    .title(Span::styled(
                        " Presets ",
                        Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                    )),
            );
            f.render_widget(empty_msg, outer[1]);
        }
    }

    fn draw_confirm_dialog(&self, f: &mut Frame) {
        let area = centered_rect(45, 25, f.area());
        let block = Block::default()
            .title(Span::styled(
                " Delete Worktree ",
                Style::default()
                    .fg(ERROR_COLOR)
                    .add_modifier(Modifier::BOLD),
            ))
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(Style::default().fg(ERROR_COLOR))
            .style(Style::default().bg(SURFACE_BG));

        let wt_name = self
            .worktrees
            .get(self.selected_worktree)
            .map(|w| w.name.clone())
            .unwrap_or_default();

        let text = Paragraph::new(vec![
            Line::from(""),
            Line::from(Span::styled(
                "Are you sure you want to delete this worktree?",
                Style::default().fg(Color::White),
            )),
            Line::from(""),
            Line::from(Span::styled(
                wt_name,
                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
            )),
            Line::from(""),
            Line::from(""),
            Line::from(vec![
                Span::styled(
                    "Y",
                    Style::default()
                        .fg(ERROR_COLOR)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled(" confirm  ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "N",
                    Style::default()
                        .fg(SUCCESS_COLOR)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled(" cancel", Style::default().fg(DIM_TEXT)),
            ]),
        ])
        .block(block)
        .alignment(Alignment::Center);
        f.render_widget(Clear, area);
        f.render_widget(text, area);
    }

    fn draw_help_overlay(&self, f: &mut Frame) {
        let area = centered_rect(65, 55, f.area());
        let block = Block::default()
            .title(Span::styled(
                " Keyboard Shortcuts ",
                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
            ))
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(Style::default().fg(ACCENT))
            .style(Style::default().bg(SURFACE_BG));

        let content = vec![
            Line::from(""),
            Line::from(Span::styled(
                "Global",
                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
            )),
            Line::from(Span::styled(
                "─".repeat(50),
                Style::default().fg(LIGHT_BORDER),
            )),
            Line::from(vec![
                Span::styled(
                    "  Tab      ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Cycle between views", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  Ctrl+C   ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Quit application", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  ?        ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Toggle this help", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(""),
            Line::from(Span::styled(
                "Worktrees View",
                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
            )),
            Line::from(Span::styled(
                "─".repeat(50),
                Style::default().fg(LIGHT_BORDER),
            )),
            Line::from(vec![
                Span::styled(
                    "  ↑/↓ j/k  ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Navigate worktrees", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  g/G      ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Jump to first/last", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  Enter    ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Focus or reopen session", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  o        ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Open in terminal", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  d        ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Toggle details sidebar", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  e        ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Expand/collapse prompt", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  x        ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Delete worktree", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  Ctrl+R   ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Refresh list", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(""),
            Line::from(Span::styled(
                "Launch & Settings",
                Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
            )),
            Line::from(Span::styled(
                "─".repeat(50),
                Style::default().fg(LIGHT_BORDER),
            )),
            Line::from(vec![
                Span::styled(
                    "  Tab      ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Move between fields", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  ←/→      ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Change preset/value", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(vec![
                Span::styled(
                    "  Enter    ",
                    Style::default()
                        .fg(Color::White)
                        .add_modifier(Modifier::BOLD),
                ),
                Span::styled("Submit/Save", Style::default().fg(DIM_TEXT)),
            ]),
            Line::from(""),
            Line::from(vec![
                Span::styled("Press ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "Esc",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" or ", Style::default().fg(DIM_TEXT)),
                Span::styled(
                    "?",
                    Style::default().fg(ACCENT).add_modifier(Modifier::BOLD),
                ),
                Span::styled(" to close", Style::default().fg(DIM_TEXT)),
            ]),
        ];

        let para = Paragraph::new(content)
            .block(block)
            .wrap(Wrap { trim: true });
        f.render_widget(Clear, area);
        f.render_widget(para, area);
    }
}

fn centered_rect(percent_x: u16, percent_y: u16, r: Rect) -> Rect {
    let popup_layout = Layout::default()
        .direction(Direction::Vertical)
        .constraints(
            [
                Constraint::Percentage((100 - percent_y) / 2),
                Constraint::Percentage(percent_y),
                Constraint::Percentage((100 - percent_y) / 2),
            ]
            .as_ref(),
        )
        .split(r);

    let vertical = Layout::default()
        .direction(Direction::Horizontal)
        .constraints(
            [
                Constraint::Percentage((100 - percent_x) / 2),
                Constraint::Percentage(percent_x),
                Constraint::Percentage((100 - percent_x) / 2),
            ]
            .as_ref(),
        )
        .split(popup_layout[1]);

    vertical[1]
}

#[cfg(test)]
mod tests {
    use super::*;
    use crossterm::event::{KeyCode, KeyEvent, KeyEventKind, KeyEventState, KeyModifiers};

    /// Helper to create a simple key event without modifiers
    fn key(code: KeyCode) -> KeyEvent {
        KeyEvent {
            code,
            modifiers: KeyModifiers::NONE,
            kind: KeyEventKind::Press,
            state: KeyEventState::NONE,
        }
    }

    /// Helper to create a key event with Ctrl modifier
    fn ctrl_key(code: KeyCode) -> KeyEvent {
        KeyEvent {
            code,
            modifiers: KeyModifiers::CONTROL,
            kind: KeyEventKind::Press,
            state: KeyEventState::NONE,
        }
    }

    /// Helper to create a key event with Shift modifier
    fn shift_key(code: KeyCode) -> KeyEvent {
        KeyEvent {
            code,
            modifiers: KeyModifiers::SHIFT,
            kind: KeyEventKind::Press,
            state: KeyEventState::NONE,
        }
    }

    // ==================== Delete Key Tests ====================

    mod delete_keys {
        use super::*;

        #[test]
        fn x_key_initiates_delete() {
            let action = map_worktree_key(key(KeyCode::Char('x')));
            assert_eq!(action, WorktreeAction::InitiateDelete);
        }

        #[test]
        fn delete_key_initiates_delete() {
            let action = map_worktree_key(key(KeyCode::Delete));
            assert_eq!(action, WorktreeAction::InitiateDelete);
        }

        #[test]
        fn backspace_key_initiates_delete() {
            let action = map_worktree_key(key(KeyCode::Backspace));
            assert_eq!(action, WorktreeAction::InitiateDelete);
        }

        #[test]
        fn all_delete_keys_produce_same_action() {
            let x_action = map_worktree_key(key(KeyCode::Char('x')));
            let delete_action = map_worktree_key(key(KeyCode::Delete));
            let backspace_action = map_worktree_key(key(KeyCode::Backspace));

            assert_eq!(x_action, delete_action);
            assert_eq!(delete_action, backspace_action);
            assert_eq!(x_action, WorktreeAction::InitiateDelete);
        }
    }

    // ==================== Navigation Key Tests ====================

    mod navigation_keys {
        use super::*;

        #[test]
        fn up_arrow_navigates_up() {
            let action = map_worktree_key(key(KeyCode::Up));
            assert_eq!(action, WorktreeAction::NavigateUp);
        }

        #[test]
        fn k_key_navigates_up() {
            let action = map_worktree_key(key(KeyCode::Char('k')));
            assert_eq!(action, WorktreeAction::NavigateUp);
        }

        #[test]
        fn down_arrow_navigates_down() {
            let action = map_worktree_key(key(KeyCode::Down));
            assert_eq!(action, WorktreeAction::NavigateDown);
        }

        #[test]
        fn j_key_navigates_down() {
            let action = map_worktree_key(key(KeyCode::Char('j')));
            assert_eq!(action, WorktreeAction::NavigateDown);
        }

        #[test]
        fn g_key_jumps_to_first() {
            let action = map_worktree_key(key(KeyCode::Char('g')));
            assert_eq!(action, WorktreeAction::JumpToFirst);
        }

        #[test]
        fn shift_g_jumps_to_last() {
            let action = map_worktree_key(key(KeyCode::Char('G')));
            assert_eq!(action, WorktreeAction::JumpToLast);
        }

        #[test]
        fn vim_navigation_keys_match_arrows() {
            assert_eq!(
                map_worktree_key(key(KeyCode::Up)),
                map_worktree_key(key(KeyCode::Char('k')))
            );
            assert_eq!(
                map_worktree_key(key(KeyCode::Down)),
                map_worktree_key(key(KeyCode::Char('j')))
            );
        }
    }

    // ==================== Confirmation Dialog Key Tests ====================

    mod confirm_delete_keys {
        use super::*;

        #[test]
        fn lowercase_y_confirms() {
            let action = map_confirm_delete_key(key(KeyCode::Char('y')));
            assert_eq!(action, ConfirmDeleteAction::Confirm);
        }

        #[test]
        fn uppercase_y_confirms() {
            let action = map_confirm_delete_key(key(KeyCode::Char('Y')));
            assert_eq!(action, ConfirmDeleteAction::Confirm);
        }

        #[test]
        fn lowercase_n_cancels() {
            let action = map_confirm_delete_key(key(KeyCode::Char('n')));
            assert_eq!(action, ConfirmDeleteAction::Cancel);
        }

        #[test]
        fn escape_cancels() {
            let action = map_confirm_delete_key(key(KeyCode::Esc));
            assert_eq!(action, ConfirmDeleteAction::Cancel);
        }

        #[test]
        fn other_keys_do_nothing() {
            assert_eq!(
                map_confirm_delete_key(key(KeyCode::Char('x'))),
                ConfirmDeleteAction::None
            );
            assert_eq!(
                map_confirm_delete_key(key(KeyCode::Enter)),
                ConfirmDeleteAction::None
            );
            assert_eq!(
                map_confirm_delete_key(key(KeyCode::Char('a'))),
                ConfirmDeleteAction::None
            );
        }

        #[test]
        fn y_and_shift_y_both_confirm() {
            let lower = map_confirm_delete_key(key(KeyCode::Char('y')));
            let upper = map_confirm_delete_key(key(KeyCode::Char('Y')));
            assert_eq!(lower, upper);
            assert_eq!(lower, ConfirmDeleteAction::Confirm);
        }
    }

    // ==================== Action Key Tests ====================

    mod action_keys {
        use super::*;

        #[test]
        fn d_toggles_details() {
            let action = map_worktree_key(key(KeyCode::Char('d')));
            assert_eq!(action, WorktreeAction::ToggleDetails);
        }

        #[test]
        fn e_toggles_prompt_expanded() {
            let action = map_worktree_key(key(KeyCode::Char('e')));
            assert_eq!(action, WorktreeAction::TogglePromptExpanded);
        }

        #[test]
        fn o_opens_worktree() {
            let action = map_worktree_key(key(KeyCode::Char('o')));
            assert_eq!(action, WorktreeAction::Open);
        }

        #[test]
        fn enter_focuses_or_opens() {
            let action = map_worktree_key(key(KeyCode::Enter));
            assert_eq!(action, WorktreeAction::FocusOrOpen);
        }

        #[test]
        fn ctrl_r_refreshes() {
            let action = map_worktree_key(ctrl_key(KeyCode::Char('r')));
            assert_eq!(action, WorktreeAction::Refresh);
        }

        #[test]
        fn ctrl_p_toggles_sidebar() {
            let action = map_worktree_key(ctrl_key(KeyCode::Char('p')));
            assert_eq!(action, WorktreeAction::ToggleSidebar);
        }
    }

    // ==================== Unhandled Key Tests ====================

    mod unhandled_keys {
        use super::*;

        #[test]
        fn unbound_letters_return_none() {
            assert_eq!(
                map_worktree_key(key(KeyCode::Char('a'))),
                WorktreeAction::None
            );
            assert_eq!(
                map_worktree_key(key(KeyCode::Char('b'))),
                WorktreeAction::None
            );
            assert_eq!(
                map_worktree_key(key(KeyCode::Char('z'))),
                WorktreeAction::None
            );
        }

        #[test]
        fn numbers_return_none() {
            assert_eq!(
                map_worktree_key(key(KeyCode::Char('1'))),
                WorktreeAction::None
            );
            assert_eq!(
                map_worktree_key(key(KeyCode::Char('9'))),
                WorktreeAction::None
            );
        }

        #[test]
        fn function_keys_return_none() {
            assert_eq!(map_worktree_key(key(KeyCode::F(1))), WorktreeAction::None);
            assert_eq!(map_worktree_key(key(KeyCode::F(12))), WorktreeAction::None);
        }

        #[test]
        fn regular_r_without_ctrl_returns_none() {
            let action = map_worktree_key(key(KeyCode::Char('r')));
            assert_eq!(action, WorktreeAction::None);
        }

        #[test]
        fn regular_p_without_ctrl_returns_none() {
            let action = map_worktree_key(key(KeyCode::Char('p')));
            assert_eq!(action, WorktreeAction::None);
        }
    }

    // ==================== Modifier Key Tests ====================

    mod modifier_keys {
        use super::*;

        #[test]
        fn ctrl_modifier_required_for_refresh() {
            let with_ctrl = map_worktree_key(ctrl_key(KeyCode::Char('r')));
            let without_ctrl = map_worktree_key(key(KeyCode::Char('r')));

            assert_eq!(with_ctrl, WorktreeAction::Refresh);
            assert_eq!(without_ctrl, WorktreeAction::None);
        }

        #[test]
        fn ctrl_modifier_required_for_sidebar() {
            let with_ctrl = map_worktree_key(ctrl_key(KeyCode::Char('p')));
            let without_ctrl = map_worktree_key(key(KeyCode::Char('p')));

            assert_eq!(with_ctrl, WorktreeAction::ToggleSidebar);
            assert_eq!(without_ctrl, WorktreeAction::None);
        }

        #[test]
        fn shift_does_not_affect_delete_keys() {
            // x with shift should still be treated as 'X' which is different from 'x'
            let shifted = map_worktree_key(shift_key(KeyCode::Char('X')));
            // Capital X is not the same as lowercase x
            assert_eq!(shifted, WorktreeAction::None);
        }
    }

    // ==================== Edge Case Tests ====================

    mod edge_cases {
        use super::*;

        #[test]
        fn tab_key_returns_none_in_worktree_action() {
            // Tab is handled at a higher level for view switching
            let action = map_worktree_key(key(KeyCode::Tab));
            assert_eq!(action, WorktreeAction::None);
        }

        #[test]
        fn escape_returns_none_in_worktree_action() {
            // Escape is handled at a higher level
            let action = map_worktree_key(key(KeyCode::Esc));
            assert_eq!(action, WorktreeAction::None);
        }

        #[test]
        fn home_and_end_keys_return_none() {
            assert_eq!(map_worktree_key(key(KeyCode::Home)), WorktreeAction::None);
            assert_eq!(map_worktree_key(key(KeyCode::End)), WorktreeAction::None);
        }

        #[test]
        fn page_up_and_down_return_none() {
            assert_eq!(
                map_worktree_key(key(KeyCode::PageUp)),
                WorktreeAction::None
            );
            assert_eq!(
                map_worktree_key(key(KeyCode::PageDown)),
                WorktreeAction::None
            );
        }
    }

    // ==================== Comprehensive Coverage Tests ====================

    mod comprehensive {
        use super::*;

        #[test]
        fn all_worktree_actions_are_reachable() {
            // Ensure every action variant can be triggered by some key
            let actions: Vec<WorktreeAction> = vec![
                map_worktree_key(key(KeyCode::Up)),
                map_worktree_key(key(KeyCode::Down)),
                map_worktree_key(key(KeyCode::Char('g'))),
                map_worktree_key(key(KeyCode::Char('G'))),
                map_worktree_key(key(KeyCode::Char('d'))),
                map_worktree_key(key(KeyCode::Char('e'))),
                map_worktree_key(key(KeyCode::Char('o'))),
                map_worktree_key(key(KeyCode::Enter)),
                map_worktree_key(key(KeyCode::Char('x'))),
                map_worktree_key(ctrl_key(KeyCode::Char('r'))),
                map_worktree_key(ctrl_key(KeyCode::Char('p'))),
                map_worktree_key(key(KeyCode::Char('z'))), // Returns None
            ];

            assert!(actions.contains(&WorktreeAction::NavigateUp));
            assert!(actions.contains(&WorktreeAction::NavigateDown));
            assert!(actions.contains(&WorktreeAction::JumpToFirst));
            assert!(actions.contains(&WorktreeAction::JumpToLast));
            assert!(actions.contains(&WorktreeAction::ToggleDetails));
            assert!(actions.contains(&WorktreeAction::TogglePromptExpanded));
            assert!(actions.contains(&WorktreeAction::Open));
            assert!(actions.contains(&WorktreeAction::FocusOrOpen));
            assert!(actions.contains(&WorktreeAction::InitiateDelete));
            assert!(actions.contains(&WorktreeAction::Refresh));
            assert!(actions.contains(&WorktreeAction::ToggleSidebar));
            assert!(actions.contains(&WorktreeAction::None));
        }

        #[test]
        fn all_confirm_actions_are_reachable() {
            let actions: Vec<ConfirmDeleteAction> = vec![
                map_confirm_delete_key(key(KeyCode::Char('y'))),
                map_confirm_delete_key(key(KeyCode::Char('n'))),
                map_confirm_delete_key(key(KeyCode::Char('a'))), // Returns None
            ];

            assert!(actions.contains(&ConfirmDeleteAction::Confirm));
            assert!(actions.contains(&ConfirmDeleteAction::Cancel));
            assert!(actions.contains(&ConfirmDeleteAction::None));
        }
    }
}
