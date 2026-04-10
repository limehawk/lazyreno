pub mod activity;
pub mod confirm;
pub mod detail;
pub mod footer;
pub mod help;
pub mod jobs;
pub mod loading;
pub mod pr_table;
pub mod repos_overlay;
pub mod sidebar;
pub mod status;
pub mod theme;

use ratatui::Frame;
use ratatui::layout::{Constraint, Direction, Layout, Rect};
use ratatui::style::Style;
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Paragraph};

use crate::app::App;
use theme::Theme;

/// Main render entry point. Draws all panels and overlays.
pub fn render(app: &App, frame: &mut Frame, theme: &Theme) {
    let size = frame.area();

    // Vertical: top bar | main content | footer
    let vert = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(4), // status top bar (bordered, 2 content lines)
            Constraint::Min(0),   // main content
            Constraint::Length(2), // footer (2-line badges)
        ])
        .split(size);
    let topbar_area = vert[0];
    let main_area = vert[1];
    let footer_area = vert[2];

    // Loading state — show centered message before first data arrives.
    if !app.loaded {
        loading::render(frame, main_area, theme);
        render_topbar(app, frame, topbar_area, theme);
        footer::render(app, frame, footer_area, theme);
        return;
    }

    // Vertical: upper (sidebar + PRs + detail) | bottom log strip
    let body = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Min(10),       // upper panels
            Constraint::Length(10),    // bottom log strip
        ])
        .split(main_area);
    let upper_area = body[0];
    let bottom_area = body[1];

    // Upper: sidebar | PRs + detail
    let cols = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Min(20),        // sidebar
            Constraint::Percentage(60), // PRs + detail
        ])
        .split(upper_area);
    let sidebar_area = cols[0];
    let middle_area = cols[1];

    // Middle vertical: PR table | detail
    let mid_v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Percentage(60), Constraint::Min(8)])
        .split(middle_area);
    let pr_table_area = mid_v[0];
    let detail_area = mid_v[1];

    // Bottom strip: activity | jobs
    let bottom_cols = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage(60), // activity (chattier)
            Constraint::Percentage(40), // jobs
        ])
        .split(bottom_area);
    let activity_area = bottom_cols[0];
    let jobs_area = bottom_cols[1];

    // Draw panels.
    render_topbar(app, frame, topbar_area, theme);
    sidebar::render(app, frame, sidebar_area, theme);
    pr_table::render(app, frame, pr_table_area, theme);
    detail::render(app, frame, detail_area, theme);
    jobs::render(app, frame, jobs_area, theme);
    activity::render(app, frame, activity_area, theme);

    // Footer bar.
    footer::render(app, frame, footer_area, theme);

    // Overlays (rendered on top).
    if app.show_help {
        let overlay = centered_rect(60, 70, size);
        help::render(app, frame, overlay, theme);
    }

    if app.show_all_repos {
        let overlay = centered_rect(50, 60, size);
        repos_overlay::render(app, frame, overlay, theme);
    }

    if app.confirming.is_some() {
        let overlay = centered_rect(40, 20, size);
        confirm::render(app, frame, overlay, theme);
    }
}

/// Two-line top bar: Renovate stats on line 1, GitHub stats on line 2.
fn render_topbar(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let sep = Span::styled(" · ", Style::default().fg(theme.dim));

    // Line 1: Renovate
    let mut rn = vec![
        Span::styled("Renovate  ", Style::default().fg(theme.muted)),
    ];
    if let Some(ref st) = app.system_status {
        rn.push(Span::styled("● ", Style::default().fg(theme.success)));
        rn.push(Span::styled(
            format!("v{}", st.version),
            Style::default().fg(theme.text),
        ));
        rn.push(sep.clone());
        rn.push(Span::styled(
            format!("Up {}", st.uptime),
            Style::default().fg(theme.muted),
        ));
        rn.push(sep.clone());
        rn.push(Span::styled(
            format!(
                "Queue: {} · Running: {} · Failed: {}",
                st.queue_size, st.running_jobs, st.failed_jobs
            ),
            Style::default().fg(theme.muted),
        ));
        if let Some(ref job) = st.last_finished {
            let dur = job
                .duration
                .map(|d| format!(" ({}s)", d.as_secs()))
                .unwrap_or_default();
            rn.push(sep.clone());
            rn.push(Span::styled(
                format!("Last: {}{}", job.repo, dur),
                Style::default().fg(theme.muted),
            ));
        }
    } else {
        rn.push(Span::styled("● ", Style::default().fg(theme.muted)));
        rn.push(Span::styled("connecting…", Style::default().fg(theme.muted)));
    }

    // Line 2: GitHub
    let mut gh = vec![
        Span::styled("GitHub    ", Style::default().fg(theme.muted)),
    ];
    match &app.github_status {
        Some(Ok(count)) => {
            gh.push(Span::styled("● ", Style::default().fg(theme.success)));
            gh.push(Span::styled(
                &app.github.owner,
                Style::default().fg(theme.text),
            ));
            gh.push(sep.clone());
            gh.push(Span::styled(
                format!("{count} repos"),
                Style::default().fg(theme.muted),
            ));
            let total_prs: usize = app.prs.values().map(|v| v.len()).sum();
            gh.push(sep.clone());
            gh.push(Span::styled(
                format!("{total_prs} open PRs"),
                Style::default().fg(theme.muted),
            ));
        }
        Some(Err(msg)) => {
            gh.push(Span::styled("● ", Style::default().fg(theme.error)));
            let hint = if msg.contains("401") {
                "bad token — check LAZYRENO_GITHUB_TOKEN"
            } else if msg.contains("403") {
                "forbidden — token may lack repo scope"
            } else if msg.contains("404") {
                "not found"
            } else {
                "fetch failed — check network"
            };
            gh.push(Span::styled(hint, Style::default().fg(theme.error)));
        }
        None => {
            gh.push(Span::styled("● ", Style::default().fg(theme.muted)));
            gh.push(Span::styled("connecting…", Style::default().fg(theme.muted)));
        }
    }

    let block = Block::default()
        .title(Span::styled(" lazyreno ", Style::default().fg(theme.accent)))
        .borders(Borders::ALL)
        .border_style(theme.border_unfocused);

    let text = vec![Line::from(rn), Line::from(gh)];
    let paragraph = Paragraph::new(text).block(block);
    frame.render_widget(paragraph, area);
}

/// Return a centered `Rect` of the given percentage size within `area`.
pub fn centered_rect(percent_x: u16, percent_y: u16, area: Rect) -> Rect {
    let v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Percentage((100 - percent_y) / 2),
            Constraint::Percentage(percent_y),
            Constraint::Percentage((100 - percent_y) / 2),
        ])
        .split(area);

    Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Percentage((100 - percent_x) / 2),
            Constraint::Percentage(percent_x),
            Constraint::Percentage((100 - percent_x) / 2),
        ])
        .split(v[1])[1]
}
