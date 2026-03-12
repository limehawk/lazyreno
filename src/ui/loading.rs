use ratatui::Frame;
use ratatui::layout::{Alignment, Rect};
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::Paragraph;

use super::theme::Theme;

/// Full-screen loading indicator shown before the first fetch completes.
pub fn render(frame: &mut Frame, area: Rect, theme: &Theme) {
    let lines = vec![
        Line::from(""),
        Line::from(Span::styled(
            "lazyreno",
            Style::default()
                .fg(theme.accent)
                .add_modifier(Modifier::BOLD),
        )),
        Line::from(""),
        Line::from(Span::styled(
            "Fetching repos, PRs, and jobs...",
            Style::default().fg(theme.muted),
        )),
    ];

    let paragraph = Paragraph::new(lines).alignment(Alignment::Center);
    // Center vertically by offsetting.
    let y_offset = area.height.saturating_sub(4) / 2;
    let centered = Rect {
        x: area.x,
        y: area.y + y_offset,
        width: area.width,
        height: 4.min(area.height),
    };
    frame.render_widget(paragraph, centered);
}
