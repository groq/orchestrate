use crate::agents;
use anyhow::{anyhow, Context, Result};
use std::process::Command;

#[derive(Debug, Clone)]
pub struct SessionInfo {
    // Agent session
    pub path: Option<String>,
    pub branch: Option<String>,
    pub agent: Option<String>,
    pub activity_log: Option<String>,

    // Custom command session
    pub command: Option<String>,
    pub title: Option<String>,
    pub color_r: Option<u8>,
    pub color_g: Option<u8>,
    pub color_b: Option<u8>,
    pub worktree_path: Option<String>,
    pub worktree_branch: Option<String>,

    pub is_custom_command: bool,
}

impl SessionInfo {
    pub fn agent_session(path: &str, branch: &str, agent: &str) -> Self {
        SessionInfo {
            path: Some(path.to_string()),
            branch: Some(branch.to_string()),
            agent: Some(agent.to_string()),
            activity_log: None,
            command: None,
            title: None,
            color_r: None,
            color_g: None,
            color_b: None,
            worktree_path: None,
            worktree_branch: None,
            is_custom_command: false,
        }
    }

    pub fn custom_command(
        cmd: &str,
        title: &str,
        color: Option<(u8, u8, u8)>,
        worktree_path: &str,
        worktree_branch: &str,
    ) -> Self {
        SessionInfo {
            path: None,
            branch: None,
            agent: None,
            activity_log: None,
            command: Some(cmd.to_string()),
            title: Some(title.to_string()),
            color_r: color.map(|c| c.0),
            color_g: color.map(|c| c.1),
            color_b: color.map(|c| c.2),
            worktree_path: Some(worktree_path.to_string()),
            worktree_branch: Some(worktree_branch.to_string()),
            is_custom_command: true,
        }
    }
}

pub struct TerminalManager {
    pub max_per_window: usize,
    pub maximize_on_launch: bool,
}

impl TerminalManager {
    pub fn new(maximize_on_launch: bool) -> Self {
        TerminalManager {
            max_per_window: 6,
            maximize_on_launch,
        }
    }

    pub fn launch_sessions(&self, sessions: &[SessionInfo], prompt: &str) -> Result<usize> {
        if sessions.is_empty() {
            return Err(anyhow!("no sessions to launch"));
        }

        let mut window_count = 0;
        for chunk in sessions.chunks(self.max_per_window) {
            let script = build_window_script(chunk, prompt);
            run_osascript(&script)?;
            if self.maximize_on_launch {
                maximize_window().ok();
            }
            window_count += 1;
        }

        Ok(window_count)
    }
}

fn osascript_escape(input: &str) -> String {
    input
        .replace('\\', "\\\\")
        .replace('"', "\\\"")
        .replace('\n', "\\n")
}

fn build_session_command(session: &SessionInfo, prompt: &str) -> String {
    if session.is_custom_command {
        build_custom_command(session)
    } else {
        build_agent_command(session, prompt)
    }
}

fn build_agent_command(session: &SessionInfo, prompt: &str) -> String {
    let title = format!(
        "{}: {}",
        session.agent.clone().unwrap_or_default(),
        session.branch.clone().unwrap_or_default()
    );
    let mut cmd_parts = vec![format!("echo -ne '\\033]0;{}\\007'", title)];

    if let Some(agent) = &session.agent {
        if let Some(color) = agents::get_color(agent) {
            cmd_parts.push(format!(
                "echo -ne '\\033]6;1;bg;red;brightness;{}\\007\\033]6;1;bg;green;brightness;{}\\007\\033]6;1;bg;blue;brightness;{}\\007'",
                color.r, color.g, color.b
            ));
        }
    }

    if let Some(path) = &session.path {
        cmd_parts.push(format!("cd \"{}\"", path));
    }

    let mut escaped_prompt = prompt.replace('\'', "'\\''");
    if escaped_prompt.is_empty() {
        escaped_prompt = prompt.to_string();
    }

    if let Some(agent) = &session.agent {
        if let Some(log) = &session.activity_log {
            cmd_parts.push(format!(
                "LOG=\"{}\"; mkdir -p \"$(dirname \\\"$LOG\\\")\"; touch \"$LOG\"; ( {{ {} '{}'; }} 2>&1 | tee -a \"$LOG\" )",
                log, agent, escaped_prompt
            ));
        } else {
            cmd_parts.push(format!("{} '{}'", agent, escaped_prompt));
        }
    }

    cmd_parts.join(" && ")
}

fn build_custom_command(session: &SessionInfo) -> String {
    let mut title = session
        .title
        .clone()
        .filter(|t| !t.is_empty())
        .unwrap_or_else(|| "terminal".to_string());

    if let Some(branch) = &session.worktree_branch {
        if !branch.is_empty() {
            title = format!("[{}] {}", branch, title);
        }
    }

    let mut cmd_parts = vec![format!("echo -ne '\\033]0;{}\\007'", title)];

    if let (Some(r), Some(g), Some(b)) = (session.color_r, session.color_g, session.color_b) {
        cmd_parts.push(format!(
            "echo -ne '\\033]6;1;bg;red;brightness;{}\\007\\033]6;1;bg;green;brightness;{}\\007\\033]6;1;bg;blue;brightness;{}\\007'",
            r, g, b
        ));
    }

    if let Some(path) = &session.worktree_path {
        cmd_parts.push(format!("cd \"{}\"", path));
    }

    if let Some(branch) = &session.worktree_branch {
        if !branch.is_empty() {
            cmd_parts.push(format!("echo 'Branch: {}'", branch));
        }
    }

    if let Some(cmd) = &session.command {
        let trimmed = cmd.trim();
        if !trimmed.is_empty() && trimmed != "\\n" {
            cmd_parts.push(trimmed.to_string());
        }
    }

    cmd_parts.join(" && ")
}

fn build_window_script(chunk: &[SessionInfo], prompt: &str) -> String {
    let mut lines = Vec::new();
    lines.push("tell application \"iTerm2\"".to_string());
    lines.push("set newWindow to (create window with default profile)".to_string());
    lines.push("set s1 to current session of newWindow".to_string());

    match chunk.len() {
        1 => {}
        2 => {
            lines.push("tell s1 to set s2 to (split vertically with default profile)".to_string());
        }
        3 => {
            lines.push("tell s1 to set s2 to (split vertically with default profile)".to_string());
            lines.push("tell s2 to set s3 to (split vertically with default profile)".to_string());
        }
        4 => {
            lines.push("tell s1 to set s2 to (split vertically with default profile)".to_string());
            lines.push("tell s1 to set s3 to (split horizontally with default profile)".to_string());
            lines.push("tell s2 to set s4 to (split horizontally with default profile)".to_string());
        }
        5 => {
            lines.push("tell s1 to set s2 to (split vertically with default profile)".to_string());
            lines.push("tell s2 to set s3 to (split vertically with default profile)".to_string());
            lines.push("tell s1 to set s4 to (split horizontally with default profile)".to_string());
            lines
                .push("tell s2 to set s5 to (split horizontally with default profile)".to_string());
        }
        _ => {
            // 6 or more (capped at 6)
            lines.push("tell s1 to set s2 to (split vertically with default profile)".to_string());
            lines.push("tell s2 to set s3 to (split vertically with default profile)".to_string());
            lines.push("tell s1 to set s4 to (split horizontally with default profile)".to_string());
            lines
                .push("tell s2 to set s5 to (split horizontally with default profile)".to_string());
            lines
                .push("tell s3 to set s6 to (split horizontally with default profile)".to_string());
        }
    }

    let pane_order = match chunk.len() {
        1 => vec!["s1"],
        2 => vec!["s1", "s2"],
        3 => vec!["s1", "s2", "s3"],
        4 => vec!["s1", "s2", "s3", "s4"],
        5 => vec!["s1", "s2", "s3", "s4", "s5"],
        _ => vec!["s1", "s2", "s3", "s4", "s5", "s6"],
    };

    for (idx, pane) in pane_order.iter().enumerate() {
        if idx >= chunk.len() {
            break;
        }
        let cmd = osascript_escape(&build_session_command(&chunk[idx], prompt));
        lines.push(format!("tell {} to write text \"{}\"", pane, cmd));
    }

    lines.push("end tell".to_string());
    lines.join("\n")
}

fn run_osascript(script: &str) -> Result<()> {
    let output = Command::new("osascript")
        .arg("-e")
        .arg(script)
        .output()
        .context("failed to run osascript")?;

    if !output.status.success() {
        return Err(anyhow!(
            "osascript failed: {}",
            String::from_utf8_lossy(&output.stderr)
        ));
    }
    Ok(())
}

pub fn focus_worktree_window(worktree_path: &str) -> Result<bool> {
    let script = format!(
        r#"
        tell application "iTerm2"
            set foundWindow to missing value
            repeat with aWindow in windows
                repeat with aTab in tabs of aWindow
                    repeat with aSession in sessions of aTab
                        try
                            set sessionPath to (tty of aSession) as text
                            if sessionPath contains "{}" then
                                set foundWindow to aWindow
                                exit repeat
                            end if
                        end try
                    end repeat
                    if foundWindow is not missing value then exit repeat
                end repeat
                if foundWindow is not missing value then exit repeat
            end repeat
            if foundWindow is not missing value then
                select foundWindow
                tell application "iTerm2" to activate
                return true
            else
                return false
            end if
        end tell
        "#,
        worktree_path
    );

    let output = Command::new("osascript").arg("-e").arg(script).output()?;
    let result = String::from_utf8_lossy(&output.stdout).trim().to_string();
    Ok(result == "true")
}

fn maximize_window() -> Result<()> {
    let script = r#"
        tell application "iTerm2"
            tell current window
                tell application "Finder"
                    set screenBounds to bounds of window of desktop
                end tell
                set bounds to {0, 25, (item 3 of screenBounds), (item 4 of screenBounds) - 50}
            end tell
        end tell
    "#;
    run_osascript(script)
}

pub fn focus_worktree_window_by_branch(branch: &str) -> Result<bool> {
    let script = format!(
        r#"
        tell application "iTerm2"
            set foundWindow to missing value
            repeat with aWindow in windows
                repeat with aTab in tabs of aWindow
                    repeat with aSession in sessions of aTab
                        try
                            set sessionName to (name of aSession) as text
                            if sessionName contains "{}" then
                                set foundWindow to aWindow
                                exit repeat
                            end if
                        end try
                    end repeat
                    if foundWindow is not missing value then exit repeat
                end repeat
                if foundWindow is not missing value then exit repeat
            end repeat
            if foundWindow is not missing value then
                select foundWindow
                tell application "iTerm2" to activate
                return true
            else
                return false
            end if
        end tell
        "#,
        branch
    );

    let output = Command::new("osascript").arg("-e").arg(script).output()?;
    let result = String::from_utf8_lossy(&output.stdout).trim().to_string();
    Ok(result == "true")
}
