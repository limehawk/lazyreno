mod api;
mod app;
mod config;
mod event;
mod keys;
mod types;
mod ui;

use std::io;
use std::panic;
use std::sync::Arc;

use anyhow::{Context, Result};
use clap::Parser;
use crossterm::{
    event::{DisableMouseCapture, EnableMouseCapture, MouseEventKind},
    execute,
    terminal::{EnterAlternateScreen, LeaveAlternateScreen, disable_raw_mode, enable_raw_mode},
};
use ratatui::Terminal;
use ratatui::backend::CrosstermBackend;
use tokio::sync::mpsc;
use tokio_util::sync::CancellationToken;
use tracing_subscriber::{EnvFilter, fmt, prelude::*};

use crate::api::fetcher::run_fetcher;
use crate::api::github::GithubClient;
use crate::api::renovate::RenovateClient;
use crate::app::App;
use crate::event::{AppEvent, EventHandler};
use crate::ui::theme::Theme;

#[derive(Parser)]
#[command(name = "lazyreno", version, about = "TUI dashboard for Renovate CE")]
struct Cli {
    /// Path to config file
    #[arg(long, short)]
    config: Option<String>,
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();

    // Logging to file — never to terminal
    let log_dir = dirs::state_dir()
        .unwrap_or_else(|| dirs::home_dir().unwrap().join(".local/state"))
        .join("lazyreno");
    std::fs::create_dir_all(&log_dir)?;
    let log_file = std::fs::File::create(log_dir.join("lazyreno.log"))?;
    tracing_subscriber::registry()
        .with(fmt::layer().with_writer(log_file).with_ansi(false))
        .with(EnvFilter::from_default_env().add_directive("lazyreno=info".parse()?))
        .init();

    // Load config
    let config_path = cli
        .config
        .map(std::path::PathBuf::from)
        .unwrap_or_else(config::Config::default_path);
    let config = config::Config::load_from_path(&config_path)
        .with_context(|| format!("Failed to load config from {}", config_path.display()))?;
    let config = Arc::new(config);

    // Init API clients
    let github = Arc::new(GithubClient::new(
        &config.github.token,
        &config.github.owner,
    )?);
    let renovate = Arc::new(RenovateClient::new(
        &config.renovate.url,
        &config.renovate.secret,
    ));

    // Channels
    let (fetch_tx, fetch_rx) = mpsc::channel(4);
    let (action_tx, action_rx) = mpsc::channel(16);
    let cancel = CancellationToken::new();

    // Spawn background fetcher
    tokio::spawn(run_fetcher(
        config.clone(),
        github.clone(),
        renovate.clone(),
        fetch_tx,
        cancel.clone(),
    ));

    // App state
    let mut app = App::new(cancel.clone(), action_tx, github, renovate);
    let theme = Theme::new(&config.ui.accent);

    // Terminal setup + run + restore
    let result = run_tui(&mut app, &theme, fetch_rx, action_rx).await;
    let _ = restore_terminal();
    result
}

async fn run_tui(
    app: &mut App,
    theme: &Theme,
    fetch_rx: mpsc::Receiver<types::FetchResult>,
    action_rx: mpsc::Receiver<types::ActionResult>,
) -> Result<()> {
    enable_raw_mode()?;
    execute!(io::stdout(), EnterAlternateScreen, EnableMouseCapture)?;

    let backend = CrosstermBackend::new(io::stdout());
    let mut terminal = Terminal::new(backend)?;

    // Panic hook to restore terminal
    let original_hook = panic::take_hook();
    panic::set_hook(Box::new(move |info| {
        let _ = restore_terminal();
        original_hook(info);
    }));

    let tick_rate = std::time::Duration::from_millis(100);
    let mut events = EventHandler::new(tick_rate, fetch_rx, action_rx);

    while app.running {
        app.clear_expired_flash();

        terminal.draw(|frame| {
            ui::render(app, frame, theme);
        })?;

        if let Some(event) = events.next().await {
            let is_scroll = matches!(
                &event,
                AppEvent::Mouse(m) if matches!(m.kind, MouseEventKind::ScrollDown | MouseEventKind::ScrollUp)
            );
            process_event(app, event);

            // Coalesce buffered scroll events so terminal multipliers
            // (e.g. Kitty's wheel_scroll_multiplier) don't cause jumps.
            if is_scroll {
                loop {
                    match events.try_next() {
                        Some(evt) if matches!(
                            &evt,
                            AppEvent::Mouse(m) if matches!(m.kind, MouseEventKind::ScrollDown | MouseEventKind::ScrollUp)
                        ) => continue,
                        Some(evt) => { process_event(app, evt); break; }
                        None => break,
                    }
                }
            }
        }
    }

    Ok(())
}

fn process_event(app: &mut App, event: AppEvent) {
    match event {
        AppEvent::Key(key) => keys::handle_key(app, key),
        AppEvent::Mouse(mouse) => keys::handle_mouse(app, mouse),
        AppEvent::FetchComplete(result) => app.apply_fetch(*result),
        AppEvent::ActionComplete(result) => app.apply_action(result),
        AppEvent::Resize(_, _) | AppEvent::Tick => {}
    }
}

fn restore_terminal() -> Result<()> {
    disable_raw_mode()?;
    execute!(io::stdout(), LeaveAlternateScreen, DisableMouseCapture)?;
    Ok(())
}
