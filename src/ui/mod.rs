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
            Constraint::Length(3), // status top bar (bordered)
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

    // Three columns: sidebar | PRs+detail | jobs+activity
    let cols = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Min(20),        // sidebar
            Constraint::Percentage(50), // PRs + detail
            Constraint::Percentage(25), // jobs + activity
        ])
        .split(main_area);
    let sidebar_area = cols[0];
    let middle_area = cols[1];
    let right_area = cols[2];

    // Middle vertical: PR table | detail
    let mid_v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Percentage(60), Constraint::Min(8)])
        .split(middle_area);
    let pr_table_area = mid_v[0];
    let detail_area = mid_v[1];

    // Right vertical: jobs | activity
    let right_v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Percentage(50), Constraint::Percentage(50)])
        .split(right_area);
    let jobs_area = right_v[0];
    let activity_area = right_v[1];

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

/// Thin status bar: "lazyreno · ● v43.55.4 · Up 16h · Queue: 0 · Running: 0"
fn render_topbar(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let mut spans = vec![
        Span::styled("lazyreno", Style::default().fg(theme.accent)),
        Span::styled(" · ", Style::default().fg(theme.dim)),
    ];

    if let Some(ref st) = app.system_status {
        spans.push(Span::styled("● ", Style::default().fg(theme.success)));
        spans.push(Span::styled(
            format!("v{}", st.version),
            Style::default().fg(theme.text),
        ));
        spans.push(Span::styled(" · ", Style::default().fg(theme.dim)));
        spans.push(Span::styled(
            format!("Up {}", st.uptime),
            Style::default().fg(theme.muted),
        ));
        spans.push(Span::styled(" · ", Style::default().fg(theme.dim)));
        spans.push(Span::styled(
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
            spans.push(Span::styled(" · ", Style::default().fg(theme.dim)));
            spans.push(Span::styled(
                format!("Last: {}{}", job.repo, dur),
                Style::default().fg(theme.muted),
            ));
        }
    } else {
        spans.push(Span::styled(
            "connecting...",
            Style::default().fg(theme.muted),
        ));
    }

    let block = Block::default()
        .title(" Renovate ")
        .borders(Borders::ALL)
        .border_style(theme.border_unfocused);

    let line = Line::from(spans);
    let paragraph = Paragraph::new(line).block(block);
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
