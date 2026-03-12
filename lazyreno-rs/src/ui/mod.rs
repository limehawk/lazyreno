pub mod theme;
pub mod sidebar;
pub mod pr_table;
pub mod detail;
pub mod status;
pub mod jobs;
pub mod help;
pub mod repos_overlay;
pub mod confirm;
pub mod flash;

use ratatui::layout::{Constraint, Direction, Layout, Rect};
use ratatui::Frame;

use crate::app::App;
use theme::Theme;

/// Main render entry point. Draws all panels and overlays.
pub fn render(app: &App, frame: &mut Frame, theme: &Theme) {
    let size = frame.area();

    // Flash bar takes 1 row at the bottom if present.
    let (main_area, flash_area) = if app.flash.is_some() {
        let chunks = Layout::default()
            .direction(Direction::Vertical)
            .constraints([Constraint::Min(0), Constraint::Length(1)])
            .split(size);
        (chunks[0], Some(chunks[1]))
    } else {
        (size, None)
    };

    // Horizontal: sidebar | right
    let h_chunks = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Min(20), Constraint::Percentage(75)])
        .split(main_area);
    let sidebar_area = h_chunks[0];
    let right_area = h_chunks[1];

    // Right vertical: PR table | bottom
    let right_v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Percentage(60), Constraint::Min(8)])
        .split(right_area);
    let pr_table_area = right_v[0];
    let bottom_area = right_v[1];

    // Bottom horizontal: detail | status_jobs
    let bottom_h = Layout::default()
        .direction(Direction::Horizontal)
        .constraints([Constraint::Ratio(1, 2), Constraint::Ratio(1, 2)])
        .split(bottom_area);
    let detail_area = bottom_h[0];
    let status_jobs_area = bottom_h[1];

    // Status_jobs vertical: status | jobs
    let sj_v = Layout::default()
        .direction(Direction::Vertical)
        .constraints([Constraint::Ratio(1, 2), Constraint::Ratio(1, 2)])
        .split(status_jobs_area);
    let status_area = sj_v[0];
    let jobs_area = sj_v[1];

    // Draw panels.
    sidebar::render(app, frame, sidebar_area, theme);
    pr_table::render(app, frame, pr_table_area, theme);
    detail::render(app, frame, detail_area, theme);
    status::render(app, frame, status_area, theme);
    jobs::render(app, frame, jobs_area, theme);

    // Flash bar.
    if let Some(area) = flash_area {
        flash::render(app, frame, area, theme);
    }

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
