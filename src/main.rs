mod agents;
mod config;
mod git;
mod launcher;
mod terminal;
mod tui;
mod util;

use crate::config::appsettings;
use crate::config::preset::{self, Config as PresetConfig};
use clap::Parser;
use std::fs;

#[derive(Parser, Debug)]
#[command(
    name = "orchestrate",
    about = "Run AI coding agents in isolated git worktrees"
)]
struct Cli {
    /// Launch the interactive TUI
    #[arg(long)]
    ui: bool,

    /// GitHub repo to clone (owner/repo)
    #[arg(long)]
    repo: Option<String>,

    /// Branch name prefix for worktrees
    #[arg(long)]
    name: Option<String>,

    /// Prompt to pass to each agent
    #[arg(long)]
    prompt: Option<String>,

    /// Preset name from settings.yaml
    #[arg(long)]
    preset: Option<String>,

    /// Multiplier for agent worktrees (overrides preset n)
    #[arg(long, default_value_t = 0)]
    n: i64,
}

fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    // Data directory
    let data_dir = util::data_dir()?;
    fs::create_dir_all(&data_dir)?;

    // Load app settings
    let (app_settings, app_settings_path) = appsettings::load_app_settings(&data_dir)?;
    if !app_settings_path.exists() {
        appsettings::save_app_settings(&data_dir, &app_settings).ok();
    }

    // Load preset config
    let preset_result = preset::load(&data_dir);
    let preset_config: Option<PresetConfig> = preset_result.config;
    let preset_path = preset_result.path.clone();

    if cli.ui {
        tui::run(data_dir.clone(), app_settings, preset_config)?;
        return Ok(());
    }

    // CLI mode requires repo, name, prompt
    let repo = cli.repo.clone().unwrap_or_default();
    let name = cli.name.clone().unwrap_or_default();
    let prompt = cli.prompt.clone().unwrap_or_default();

    if repo.is_empty() || name.is_empty() || prompt.is_empty() {
        print_usage();
        std::process::exit(1);
    }

    if preset_config.is_none() {
        eprintln!(
            "Error: settings.yaml not found. Please create one in {}",
            util::display_path(data_dir.join("settings.yaml"))
        );
        std::process::exit(1);
    }

    println!(
        "Settings: {}",
        preset_path
            .as_ref()
            .map(|p| util::display_path(p))
            .unwrap_or_else(|| "not found".to_string())
    );
    println!("App Settings: {}", util::display_path(app_settings_path));

    let default_preset_name = if let Some(cfg) = preset_config.as_ref() {
        preset::get_default_preset_name(cfg, &app_settings.session.default_preset)
    } else {
        app_settings.session.default_preset.clone()
    };

    let preset_name = cli.preset.clone().unwrap_or(default_preset_name);

    let preset = preset_config
        .as_ref()
        .and_then(|cfg| preset::get_preset(cfg, &preset_name))
        .unwrap_or_else(|| {
            eprintln!(
                "Warning: preset '{}' not found, using single droid agent",
                preset_name
            );
            vec![preset::Worktree {
                agent: "droid".to_string(),
                n: 1,
                commands: vec![],
            }]
        });

    let multiplier = if cli.n > 0 { cli.n } else { 1 };

    let opts = launcher::Options {
        repo: repo.clone(),
        name: name.clone(),
        prompt: prompt.clone(),
        preset_name: preset_name.clone(),
        multiplier,
        data_dir: data_dir.clone(),
        preset,
        maximize_on_launch: app_settings.terminal.maximize_on_launch,
    };

    println!("Repo: {}", repo);
    println!("Fetching latest from main branch...");
    match launcher::launch(opts) {
        Ok(res) => {
            println!(
                "Started {} session(s) in {} worktree(s)!",
                res.sessions.len(),
                res.terminal_window_count
            );
        }
        Err(err) => {
            eprintln!("Error: {}", err);
            std::process::exit(1);
        }
    }

    Ok(())
}

fn print_usage() {
    println!("Usage: orchestrate [options]");
    println!();
    println!("Modes:");
    println!("  --ui              Launch the interactive TUI to manage settings");
    println!();
    println!("CLI Mode (requires --repo, --name, --prompt):");
    println!("  --repo     GitHub repo to clone (owner/repo)");
    println!("  --name     Branch name prefix for worktrees");
    println!("  --prompt   Prompt to pass to each agent");
    println!("  --preset   Preset name from settings.yaml (optional)");
    println!("  --n        Multiplier for agent worktrees (optional)");
    println!();
    println!("Examples:");
    println!("  orchestrate --ui");
    println!("  orchestrate --repo owner/repo --name feature --prompt 'Fix bug'");
}
