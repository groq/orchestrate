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
use crossterm::terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen};
use crossterm::ExecutableCommand;
use ratatui::backend::CrosstermBackend;
use ratatui::layout::{Alignment, Constraint, Direction, Layout, Rect};
use ratatui::style::{Color, Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{
    Block, BorderType, Borders, Cell, Clear, Gauge, List, ListItem, Paragraph, Row, Sparkline, Table, Wrap,
};
use ratatui::Terminal;
use std::fs;
use std::path::PathBuf;
use std::time::{Duration, Instant};

type Frame<'a> = ratatui::Frame<'a>;

pub fn run(data_dir: PathBuf, app_settings: AppSettings, preset_config: Option<PresetConfig>) -> Result<()> {
    enable_raw_mode()?;
    let mut stdout = std::io::stdout();
    stdout.execute(EnterAlternateScreen)?;
    stdout.execute(Hide)?;

    let backend = CrosstermBackend::new(stdout);
    let mut terminal = Terminal::new(backend)?;

    let mut app = App::new(data_dir, app_settings, preset_config);
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
    Theme,
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
        self.agents.get(0).cloned().unwrap_or_else(|| "-".to_string())
    }
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
            SettingsField::Maximize => SettingsField::Theme,
            SettingsField::Theme => SettingsField::DefaultPreset,
            SettingsField::DefaultPreset => SettingsField::AutoClean,
            SettingsField::AutoClean => SettingsField::RetentionDays,
            SettingsField::RetentionDays => SettingsField::TerminalType,
        }
    }

    fn prev_field(&mut self) {
        self.focused = match self.focused {
            SettingsField::TerminalType => SettingsField::RetentionDays,
            SettingsField::Maximize => SettingsField::TerminalType,
            SettingsField::Theme => SettingsField::Maximize,
            SettingsField::DefaultPreset => SettingsField::Theme,
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
    should_quit: bool,

    // Worktrees
    worktrees: Vec<WorktreeItem>,
    worktrees_loading: bool,
    selected_worktree: usize,
    show_details: bool,
    confirming_delete: bool,
    last_refresh: Instant,

    // Launch
    launch_form: LaunchForm,

    // Settings
    settings_form: SettingsForm,
}

impl App {
    fn new(data_dir: PathBuf, app_settings: AppSettings, preset_config: Option<PresetConfig>) -> Self {
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

        App {
            data_dir,
            app_settings,
            preset_config,
            view: View::Worktrees,
            sidebar_open: false,
            status: None,
            help_expanded: false,
            should_quit: false,
            worktrees: Vec::new(),
            worktrees_loading: false,
            selected_worktree: 0,
            show_details: false,
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
                let last_commit = git::last_commit_time(&wt_path).unwrap_or_else(|_| "-".to_string());
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
                    Some(self.data_dir.join("activity").join(format!("{}.log", branch)))
                } else {
                    None
                };
                let (activity_recent, activity_active) = if let Some(path) = &activity_path {
                    let lines = util::tail_lines(path, 6).unwrap_or_default();
                    let active = util::modified_within(path, 10).unwrap_or(false);
                    (lines, active)
                } else {
                    (vec![], false)
                };

                let meta = session::load_session_metadata(&wt_path).ok();
                let (has_meta, created_at, prompt, preset_name, agents, repo) = if let Some(m) = meta {
                    (
                        true,
                        Some(m.created_at.with_timezone(&Local)),
                        m.prompt,
                        m.preset_name,
                        m.agents,
                        if repo_short.is_empty() { m.repo } else { repo_short.clone() },
                    )
                } else {
                    (false, None, "".to_string(), "".to_string(), vec![], repo_short.clone())
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
                            .map(|t| DateTime::<Local>::from(t))
                            .unwrap_or_else(|| Local::now())
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
                wt.activity_recent = util::tail_lines(path, 10).unwrap_or_default();
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

        // Quick view selection via number keys
        match key.code {
            KeyCode::Char('1') => {
                self.view = View::Worktrees;
                return Ok(true);
            }
            KeyCode::Char('2') => {
                self.view = View::Launch;
                return Ok(true);
            }
            KeyCode::Char('3') => {
                self.view = View::Settings;
                return Ok(true);
            }
            KeyCode::Char('4') => {
                self.view = View::Presets;
                return Ok(true);
            }
            _ => {}
        }

        if key.code == KeyCode::Char('?') || (key.code == KeyCode::Char('/') && key.modifiers.contains(KeyModifiers::SHIFT)) {
            self.help_expanded = !self.help_expanded;
            return Ok(true);
        }
        if self.help_expanded && key.code == KeyCode::Esc {
            self.help_expanded = false;
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
            match key.code {
                KeyCode::Char('y') | KeyCode::Char('Y') => {
                    self.delete_selected_worktree()?;
                    self.confirming_delete = false;
                    return Ok(true);
                }
                KeyCode::Char('n') | KeyCode::Esc => {
                    self.confirming_delete = false;
                    return Ok(true);
                }
                _ => return Ok(false),
            }
        }

        match key.code {
            KeyCode::Char('p') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.sidebar_open = !self.sidebar_open;
                return Ok(true);
            }
            KeyCode::Char('r') if key.modifiers.contains(KeyModifiers::CONTROL) => {
                self.refresh_worktrees()?;
                return Ok(true);
            }
            KeyCode::Up | KeyCode::Char('k') => {
                if self.selected_worktree > 0 {
                    self.selected_worktree -= 1;
                }
                return Ok(true);
            }
            KeyCode::Down | KeyCode::Char('j') => {
                if self.selected_worktree + 1 < self.worktrees.len() {
                    self.selected_worktree += 1;
                }
                return Ok(true);
            }
            KeyCode::Char('g') => {
                self.selected_worktree = 0;
                return Ok(true);
            }
            KeyCode::Char('G') => {
                if !self.worktrees.is_empty() {
                    self.selected_worktree = self.worktrees.len() - 1;
                }
                return Ok(true);
            }
            KeyCode::Char('d') => {
                self.show_details = !self.show_details;
                self.sidebar_open = self.show_details;
                return Ok(true);
            }
            KeyCode::Char('o') => {
                self.reopen_selected_worktree()?;
                return Ok(true);
            }
            KeyCode::Enter => {
                let focused = self.focus_selected_worktree()?;
                if !focused {
                    self.reopen_selected_worktree()?;
                }
                return Ok(true);
            }
            KeyCode::Char('x') | KeyCode::Delete => {
                self.confirming_delete = true;
                return Ok(true);
            }
            _ => {}
        }
        Ok(false)
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
                if key.modifiers.contains(KeyModifiers::CONTROL) || self.launch_form.focused == LaunchField::Launch {
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
                    self.settings_form
                        .app_settings
                        .session
                        .default_preset
                        .pop();
                    return Ok(true);
                }
            }
            KeyCode::Char(c) => {
                if self.settings_form.focused == SettingsField::DefaultPreset {
                    self.settings_form.app_settings.session.default_preset.push(c);
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
                self.settings_form.app_settings.terminal.r#type = match self.settings_form.app_settings.terminal.r#type {
                    TerminalType::ITerm2 => {
                        if delta > 0 { TerminalType::Terminal } else { TerminalType::ITerm2 }
                    }
                    TerminalType::Terminal => {
                        if delta > 0 { TerminalType::ITerm2 } else { TerminalType::Terminal }
                    }
                }
            }
            SettingsField::Maximize => {
                if delta != 0 {
                    self.settings_form.app_settings.terminal.maximize_on_launch =
                        !self.settings_form.app_settings.terminal.maximize_on_launch;
                }
            }
            SettingsField::Theme => {
                let themes = appsettings::theme_options();
                if let Some(idx) = themes
                    .iter()
                    .position(|t| *t == self.settings_form.app_settings.ui.theme)
                {
                    let mut new_idx = idx as i32 + delta;
                    if new_idx < 0 {
                        new_idx = themes.len() as i32 - 1;
                    }
                    if new_idx >= themes.len() as i32 {
                        new_idx = 0;
                    }
                    self.settings_form.app_settings.ui.theme = themes[new_idx as usize].to_string();
                }
            }
            SettingsField::DefaultPreset => {
                if let Some(cfg) = self.preset_config.as_ref() {
                    let mut names = cfg.presets.keys().cloned().collect::<Vec<_>>();
                    names.sort();
                    if !names.is_empty() {
                        let current = self.settings_form.app_settings.session.default_preset.clone();
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
                let mut days = self.settings_form.app_settings.session.worktree_retention_days;
                days = (days as i32 + delta).clamp(1, 365) as i64;
                self.settings_form.app_settings.session.worktree_retention_days = days;
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
                .unwrap_or_else(|| vec![preset::Worktree {
                    agent: "claude".to_string(),
                    n: 1,
                    commands: vec![],
                }]);
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

            let mgr = terminal::TerminalManager::new(self.app_settings.terminal.maximize_on_launch);
            match mgr.launch_sessions(&sessions, "") {
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
        let size = f.size();
        let mut constraints = vec![Constraint::Length(3), Constraint::Min(0)];
        if self.app_settings.ui.show_status_bar {
            constraints.push(Constraint::Length(1));
        }
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints(constraints)
            .split(size);

        self.draw_header(f, chunks[0]);

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
        let tabs = vec!["Worktrees", "Launch", "Settings", "Presets"];
        let spans: Vec<Span> = tabs
            .iter()
            .enumerate()
            .map(|(idx, label)| {
                let active = match (self.view, idx) {
                    (View::Worktrees, 0) => true,
                    (View::Launch, 1) => true,
                    (View::Settings, 2) => true,
                    (View::Presets, 3) => true,
                    _ => false,
                };
                if active {
                    Span::styled(
                        format!(" {} ", label),
                        Style::default()
                            .fg(Color::Black)
                            .bg(Color::Cyan)
                            .add_modifier(Modifier::BOLD),
                    )
                } else {
                    Span::styled(
                        format!(" {} ", label),
                        Style::default()
                            .fg(Color::Gray)
                            .bg(Color::DarkGray),
                    )
                }
            })
            .collect();
        let line = Line::from(spans);
        let mut title_line = Span::styled(
            "Orchestrate",
            Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD),
        );
        let nav_hint = Span::styled(
            " 1:Worktrees • 2:Launch • 3:Settings • 4:Presets • Tab/Shift+Tab cycle ",
            Style::default().fg(Color::Gray),
        );
        if !self.app_settings.ui.show_status_bar {
            if let Some((msg, is_err)) = &self.status {
                let color = if *is_err { Color::Red } else { Color::Green };
                title_line = Span::styled(
                    format!("Orchestrate — {}", msg),
                    Style::default().fg(color).add_modifier(Modifier::BOLD),
                );
            }
        }
        let block = Block::default()
            .borders(Borders::BOTTOM)
            .border_type(BorderType::Rounded)
            .title(title_line)
            .title_bottom(nav_hint);
        let paragraph = Paragraph::new(line).block(block);
        f.render_widget(paragraph, area);
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
        let (msg, color) = if let Some((msg, is_error)) = &self.status {
            let c = if *is_error { Color::Red } else { Color::Green };
            (msg.clone(), c)
        } else {
            ("Ctrl+C to quit • Tab to switch views".to_string(), Color::Gray)
        };
        let block = Block::default().borders(Borders::TOP);
        let text = Paragraph::new(msg).style(Style::default().fg(color)).block(block);
        f.render_widget(text, area);
    }

    fn draw_worktrees(&self, f: &mut Frame, area: Rect) {
        if self.worktrees_loading {
            let text = Paragraph::new("Loading worktrees...").style(Style::default().fg(Color::Cyan));
            f.render_widget(text, area);
            return;
        }
        if self.worktrees.is_empty() {
            let msg = "No worktrees found. Use the launch view or CLI to create sessions.";
            let text = Paragraph::new(msg).style(Style::default().fg(Color::Gray));
            f.render_widget(text, area);
            return;
        }

        let total_adds: i32 = self.worktrees.iter().map(|w| w.adds).sum();
        let total_dels: i32 = self.worktrees.iter().map(|w| w.deletes).sum();
        let selected = self.worktrees.get(self.selected_worktree);

        let layout = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Length(7), Constraint::Min(0)].as_ref())
            .split(area);

        // Metrics bar inspired by ratatui docs components
        let adds_bar = Gauge::default()
            .block(
                Block::default()
                    .title("Adds")
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded),
            )
            .gauge_style(Style::default().fg(Color::Green).bg(Color::Black))
            .ratio(if total_adds + total_dels == 0 {
                0.0
            } else {
                (total_adds as f64) / (total_adds + total_dels) as f64
            });
        let dels_bar = Gauge::default()
            .block(
                Block::default()
                    .title("Deletes")
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded),
            )
            .gauge_style(Style::default().fg(Color::Red).bg(Color::Black))
            .ratio(if total_adds + total_dels == 0 {
                0.0
            } else {
                (total_dels as f64) / (total_adds + total_dels) as f64
            });
        let spark_data = selected
            .map(|w| vec![w.adds as u64, w.deletes as u64])
            .unwrap_or_else(|| vec![0, 0]);
        let spark = Sparkline::default()
            .block(
                Block::default()
                    .title("Selected (+/-)")
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded),
            )
            .data(&spark_data)
            .style(Style::default().fg(Color::Cyan));
        let totals = Paragraph::new(Line::from(vec![
            Span::styled(format!("Worktrees: {}", self.worktrees.len()), Style::default().fg(Color::Cyan)),
            Span::raw("   "),
            Span::styled(
                format!(
                    "Dirty: {}",
                    self.worktrees.iter().filter(|w| w.adds + w.deletes > 0).count()
                ),
                Style::default().fg(Color::Yellow),
            ),
            Span::raw("   "),
            Span::styled(
                format!("With metadata: {}", self.worktrees.iter().filter(|w| w.has_meta).count()),
                Style::default().fg(Color::Green),
            ),
        ]))
        .block(
            Block::default()
                .borders(Borders::ALL)
                .border_type(BorderType::Rounded)
                .title("Overview"),
        );

        let metrics = Layout::default()
            .direction(Direction::Horizontal)
            .constraints(
                [
                    Constraint::Percentage(25),
                    Constraint::Percentage(25),
                    Constraint::Percentage(25),
                    Constraint::Percentage(25),
                ]
                .as_ref(),
            )
            .split(layout[0]);
        f.render_widget(adds_bar, metrics[0]);
        f.render_widget(dels_bar, metrics[1]);
        f.render_widget(spark, metrics[2]);
        f.render_widget(totals, metrics[3]);

        // Clean layout: summary table, left (commits + files), right full-height activity, then table
        let sections = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Length(5), Constraint::Length(12), Constraint::Min(0)].as_ref())
            .split(area);

        // Summary as table
        let summary_rows = if let Some(wt) = selected {
            vec![
                Row::new(vec![Cell::from("Repo"), Cell::from(wt.repo.clone())]),
                Row::new(vec![Cell::from("Branch"), Cell::from(wt.branch.clone())]),
                Row::new(vec![Cell::from("Agent"), Cell::from(if wt.agents.is_empty() { "-".to_string() } else { wt.agents.join(", ") })]),
                Row::new(vec![Cell::from("Δ"), Cell::from(format!("+{} / -{}", wt.adds, wt.deletes))]),
            ]
        } else {
            vec![
                Row::new(vec![Cell::from("Repo"), Cell::from("-")]),
                Row::new(vec![Cell::from("Branch"), Cell::from("-")]),
                Row::new(vec![Cell::from("Agent"), Cell::from("-")]),
                Row::new(vec![Cell::from("Δ"), Cell::from("-")]),
            ]
        };
        let summary = Table::new(summary_rows, [Constraint::Length(8), Constraint::Min(0)])
            .block(
                Block::default()
                    .title("Current • Enter: focus→open • o: open • x: delete • Ctrl+R: refresh")
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded),
            )
            .column_spacing(1);
        f.render_widget(summary, sections[0]);

        // Middle row split: left (commits + files) and right (activity stream, full height)
        let mid = Layout::default()
            .direction(Direction::Horizontal)
            .constraints([Constraint::Percentage(65), Constraint::Percentage(35)].as_ref())
            .split(sections[1]);

        let left_mid = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Percentage(50), Constraint::Percentage(50)].as_ref())
            .split(mid[0]);

        let mut commits_lines = vec![];
        if let Some(wt) = selected {
            if wt.recent_commits.is_empty() {
                commits_lines.push("No recent commits".to_string());
            } else {
                commits_lines.extend(wt.recent_commits.iter().cloned());
            }
        }
        let commits = Paragraph::new(commits_lines.join("\n"))
            .block(
                Block::default()
                    .title("Recent Commits")
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded),
            )
            .wrap(Wrap { trim: false });
        f.render_widget(commits, left_mid[0]);

        let mut files_lines = vec![];
        if let Some(wt) = selected {
            if wt.file_stats.is_empty() {
                files_lines.push("No uncommitted changes".to_string());
            } else {
                for fs in wt.file_stats.iter().take(10) {
                    files_lines.push(format!("+{} -{} {}", fs.adds, fs.deletes, fs.path));
                }
            }
        }
        let files = Paragraph::new(files_lines.join("\n"))
            .block(
                Block::default()
                    .title("Files Changed")
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded),
            )
            .wrap(Wrap { trim: false });
        f.render_widget(files, left_mid[1]);

        let mut activity_lines = vec![];
        if let Some(wt) = selected {
            let status = if wt.activity_active { "● Active" } else { "● Idle" };
            let color = if wt.activity_active { Color::Green } else { Color::Red };
            activity_lines.push(status.to_string());
            if let Some(p) = &wt.activity_log {
                activity_lines.push(format!("Log: {}", util::display_path(p)));
            }
            if wt.activity_recent.is_empty() {
                activity_lines.push("No recent output".to_string());
            } else {
                activity_lines.extend(wt.activity_recent.iter().cloned());
            }
            let activity = Paragraph::new(activity_lines.join("\n"))
                .style(Style::default().fg(color))
                .block(
                    Block::default()
                        .title("Agent Activity")
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded),
                )
                .wrap(Wrap { trim: false });
            f.render_widget(activity, mid[1]);
        } else {
            let activity = Paragraph::new("No selection")
                .block(
                    Block::default()
                        .title("Agent Activity")
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded),
                )
                .wrap(Wrap { trim: false });
            f.render_widget(activity, mid[1]);
        }

        let rows: Vec<Row> = self
            .worktrees
            .iter()
            .enumerate()
            .map(|(idx, wt)| {
                let changes = if wt.adds == 0 && wt.deletes == 0 {
                    "-".to_string()
                } else {
                    format!("+{} / -{}", wt.adds, wt.deletes)
                };
                let status_dot = if wt.activity_active { "●" } else { "●" };
                let status_color = if wt.activity_active { Color::Green } else { Color::Red };
                let style = if idx == self.selected_worktree {
                    Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD)
                } else {
                    Style::default()
                };
                Row::new(vec![
                    Span::styled(status_dot, Style::default().fg(status_color)).to_string(),
                    wt.name.clone(),
                    wt.repo.clone(),
                    wt.branch.clone(),
                    changes,
                    wt.display_agent(),
                    wt.last_commit.clone(),
                ])
                .style(style)
            })
            .collect();

        let widths = [
            Constraint::Length(3),
            Constraint::Percentage(23),
            Constraint::Percentage(22),
            Constraint::Percentage(15),
            Constraint::Percentage(12),
            Constraint::Percentage(10),
            Constraint::Percentage(15),
        ];
        let table = Table::new(rows, widths)
            .header(
                Row::new(vec![" ", "Name", "Repo", "Branch", "Changes", "Agent", "Last Commit"])
                    .style(Style::default().fg(Color::Gray)),
            )
            .column_spacing(2)
            .block(
                Block::default()
                    .borders(Borders::ALL)
                    .border_type(BorderType::Rounded)
                    .title("Worktrees"),
            );
        f.render_widget(table, layout[1]);
    }

    fn draw_sidebar(&self, f: &mut Frame, area: Rect) {
        if !self.show_details {
            let text = Paragraph::new("Press d to toggle details").wrap(Wrap { trim: true }).block(
                Block::default()
                    .borders(Borders::ALL)
                    .title("Sidebar"),
            );
            f.render_widget(text, area);
            return;
        }
        if let Some(wt) = self.worktrees.get(self.selected_worktree) {
            let mut lines: Vec<Line> = Vec::new();
            lines.push(Line::from(vec![Span::styled(
                wt.name.clone(),
                Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD),
            )]));
            lines.push(Line::from(format!("Path: {}", util::display_path(&wt.path))));
            lines.push(Line::from(format!("Repo: {}", wt.repo)));
            lines.push(Line::from(format!("Branch: {}", wt.branch)));
            if wt.has_meta {
                lines.push(Line::from(format!(
                    "Preset: {}",
                    if wt.preset_name.is_empty() { "-" } else { &wt.preset_name }
                )));
            }
            if !wt.agents.is_empty() {
                lines.push(Line::from(format!("Agents: {}", wt.agents.join(", "))));
            }
            if let Some(created) = wt.created_at {
                lines.push(Line::from(format!("Created: {}", created.format("%Y-%m-%d %H:%M"))));
            }
            lines.push(Line::from(format!("Last commit: {}", wt.last_commit)));
            lines.push(Line::from(format!("Adds/Deletes: +{} / -{}", wt.adds, wt.deletes)));
            if !wt.prompt.is_empty() {
                lines.push(Line::from("Prompt:"));
                lines.push(Line::from(wt.prompt.clone()));
            }
            if !wt.file_stats.is_empty() {
                lines.push(Line::from("Files changed:"));
                for fs in wt.file_stats.iter().take(8) {
                    lines.push(Line::from(format!("+{} -{} {}", fs.adds, fs.deletes, fs.path)));
                }
            }
            let text = Paragraph::new(lines)
                .block(
                    Block::default()
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded)
                        .title("Details"),
                )
                .wrap(Wrap { trim: false });
            f.render_widget(text, area);
        }
    }

    fn draw_launch(&self, f: &mut Frame, area: Rect) {
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints(
                [
                    Constraint::Length(3),
                    Constraint::Length(3),
                    Constraint::Length(7),
                    Constraint::Length(3),
                    Constraint::Length(3),
                    Constraint::Min(0),
                ]
                .as_ref(),
            )
            .split(area);

        let repo = Paragraph::new(self.launch_form.repo.as_str())
            .block(self.input_block("Repository", self.launch_form.focused == LaunchField::Repo, "owner/repo"))
            .wrap(Wrap { trim: true });
        f.render_widget(repo, chunks[0]);

        let name = Paragraph::new(self.launch_form.name.as_str())
            .block(self.input_block("Branch prefix", self.launch_form.focused == LaunchField::Name, "feature-name"))
            .wrap(Wrap { trim: true });
        f.render_widget(name, chunks[1]);

        let prompt = Paragraph::new(self.launch_form.prompt.as_str())
            .block(self.input_block("Prompt", self.launch_form.focused == LaunchField::Prompt, "What should the agent do?"))
            .wrap(Wrap { trim: true });
        f.render_widget(prompt, chunks[2]);

        let preset = {
            let text = if self.launch_form.preset_names.is_empty() {
                "default".to_string()
            } else {
                let count = self.launch_form.preset_names.len();
                format!(
                    "< {} >   ({}/{})",
                    self.launch_form.preset(),
                    self.launch_form.preset_idx + 1,
                    count
                )
            };
            Paragraph::new(text)
                .block(self.input_block("Preset (←/→)", self.launch_form.focused == LaunchField::Preset, ""))
                .wrap(Wrap { trim: true })
        };
        f.render_widget(preset, chunks[3]);

        let btn_block = Block::default()
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(if self.launch_form.focused == LaunchField::Launch {
                Style::default().fg(Color::Cyan)
            } else {
                Style::default()
            })
            .title("Launch (Enter)");
        let btn = Paragraph::new("Launch sessions").block(btn_block).alignment(Alignment::Center);
        f.render_widget(btn, chunks[4]);

        let help = Paragraph::new("Tab to move • Ctrl+Enter or Launch to start • Prompt accepts newlines")
            .style(Style::default().fg(Color::Gray));
        f.render_widget(help, chunks[5]);
    }

    fn input_block(&self, label: &str, focused: bool, hint: &str) -> Block<'static> {
        let title = if hint.is_empty() {
            label.to_string()
        } else {
            format!("{} — {}", label, hint)
        };
        Block::default()
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(if focused { Style::default().fg(Color::Cyan) } else { Style::default() })
            .title(title)
    }

    fn draw_settings(&self, f: &mut Frame, area: Rect) {
        let items = vec![
            (
                "Terminal Type",
                format!("{:?}", self.settings_form.app_settings.terminal.r#type),
                self.settings_form.focused == SettingsField::TerminalType,
            ),
            (
                "Maximize on Launch",
                self.settings_form.app_settings.terminal.maximize_on_launch.to_string(),
                self.settings_form.focused == SettingsField::Maximize,
            ),
            (
                "Theme",
                self.settings_form.app_settings.ui.theme.clone(),
                self.settings_form.focused == SettingsField::Theme,
            ),
            (
                "Default Preset",
                self.settings_form.app_settings.session.default_preset.clone(),
                self.settings_form.focused == SettingsField::DefaultPreset,
            ),
            (
                "Auto Clean Worktrees",
                self.settings_form.app_settings.session.auto_clean_worktrees.to_string(),
                self.settings_form.focused == SettingsField::AutoClean,
            ),
            (
                "Retention Days",
                self.settings_form.app_settings.session.worktree_retention_days.to_string(),
                self.settings_form.focused == SettingsField::RetentionDays,
            ),
        ];

        let rows: Vec<Row> = items
            .into_iter()
            .map(|(label, value, focused)| {
                let style = if focused {
                    Style::default().fg(Color::Cyan).add_modifier(Modifier::BOLD)
                } else {
                    Style::default()
                };
                Row::new(vec![label.to_string(), value]).style(style)
            })
            .collect();

        let table = Table::new(
            rows,
            [Constraint::Percentage(50), Constraint::Percentage(50)],
        )
        .header(Row::new(vec!["Setting", "Value"]).style(Style::default().fg(Color::Gray)))
        .block(
            Block::default()
                .borders(Borders::ALL)
                .border_type(BorderType::Rounded)
                .title("Settings (Enter to save, ←/→ to toggle)"),
        );
        f.render_widget(table, area);
    }

    fn draw_presets(&self, f: &mut Frame, area: Rect) {
        if let Some(cfg) = &self.preset_config {
            let mut items = Vec::new();
            for (name, preset) in cfg.presets.iter() {
                let mut lines = vec![format!("Preset: {}", name)];
                for (idx, wt) in preset.iter().enumerate() {
                    lines.push(format!("  {}. Agent: {}", idx + 1, wt.agent));
                    for cmd in &wt.commands {
                        let title = cmd.display_title();
                        lines.push(format!("     - {}", title));
                    }
                }
                items.push(ListItem::new(lines.join("\n")));
            }
            let list = List::new(items)
                .block(
                    Block::default()
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded)
                        .title("Presets"),
                )
                .highlight_style(Style::default().fg(Color::Cyan));
            f.render_widget(list, area);
        } else {
            let text = Paragraph::new("No presets found. Create settings.yaml in your data dir.")
                .style(Style::default().fg(Color::Gray))
                .block(
                    Block::default()
                        .borders(Borders::ALL)
                        .border_type(BorderType::Rounded)
                        .title("Presets"),
                );
            f.render_widget(text, area);
        }
    }

    fn draw_confirm_dialog(&self, f: &mut Frame) {
        let area = centered_rect(50, 30, f.size());
        let block = Block::default()
            .title("Delete worktree?")
            .borders(Borders::ALL)
            .border_style(Style::default().fg(Color::Red));
        let text = Paragraph::new("Press Y to confirm, N to cancel").block(block).alignment(Alignment::Center);
        f.render_widget(Clear, area);
        f.render_widget(text, area);
    }

    fn draw_help_overlay(&self, f: &mut Frame) {
        let area = centered_rect(70, 60, f.size());
        let bindings = vec![
            "Global: Tab cycle views • Ctrl+C quit • ? toggle help",
            "Worktrees: ↑/↓/g/G navigate • Enter focus • o open • d details • x delete • Ctrl+R refresh • Ctrl+P toggle sidebar",
            "Launch: Tab/Shift+Tab move • Ctrl+Enter launch • arrows change preset",
            "Settings: ↑/↓ select • ←/→ toggle values • Enter save",
        ];
        let content = bindings.join("\n");
        let block = Block::default()
            .title("Help & Shortcuts")
            .borders(Borders::ALL)
            .border_type(BorderType::Rounded)
            .border_style(Style::default().fg(Color::Cyan));
        let para = Paragraph::new(content)
            .style(Style::default().fg(Color::White))
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
