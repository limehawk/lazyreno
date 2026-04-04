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

use crate::app::App;
use theme::Theme;

/// Main render entry point. Draws all panels and overlays.
pub fn render(app: &App, frame: &mut Frame, theme: &Theme) {
    let size = frame.area();

    // Bottom bar: footer only. Activity log replaced the flash bar.
    let vert = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Min(0), Constraint::Length(1)])
        .split(size);
    let main_area = vert[0];
    let footer_area = vert[1];

    // Loading state — show centered message before first data arrives.
    if !app.loaded {
        loading::render(frame, main_area, theme);
        footer::render(app, frame, footer_area, theme);
        return;
    }

    // Horizontal: sidebar | middle (PRs+detail) | right (system+jobs)
    let cols = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([
            Constraint::Min(20),        // sidebar
            Constraint::Percentage(50), // PRs + detail
            Constraint::Percentage(25), // system + jobs
        ])
        .split(main_area);
    let sidebar_area = cols[0];
    let middle_area = cols[1];
    let right_area = cols[2];

    // Middle vertical: PR table | detail (equal height)
    let mid_v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Percentage(60), Constraint::Min(8)])
        .split(middle_area);
    let pr_table_area = mid_v[0];
    let detail_area = mid_v[1];

    // Right vertical: status | jobs | activity (activity matches detail height)
    let detail_height = detail_area.height;
    let right_v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(5),            // status
            Constraint::Min(5),              // jobs (takes remaining)
            Constraint::Length(detail_height), // activity = detail height
        ])
        .split(right_area);
    let status_area = right_v[0];
    let jobs_area = right_v[1];
    let activity_area = right_v[2];

    // Draw panels.
    sidebar::render(app, frame, sidebar_area, theme);
    pr_table::render(app, frame, pr_table_area, theme);
    detail::render(app, frame, detail_area, theme);
    status::render(app, frame, status_area, theme);
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
