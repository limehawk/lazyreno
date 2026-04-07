use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Paragraph};

use super::theme::Theme;
use crate::app::App;
use crate::types::{Panel, UpdateType};

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let focused = app.focused_panel == Panel::Detail;
    let border_style = if focused {
        theme.border_focused
    } else {
        theme.border_unfocused
    };

    let block = Block::default()
        .title(" Detail ")
        .borders(Borders::ALL)
        .border_style(border_style);

    let content = if let Some(pr) = app.selected_pr() {
        let type_style = match pr.update_type {
            UpdateType::Major => Style::default().fg(theme.major),
            UpdateType::Minor | UpdateType::Patch => Style::default().fg(theme.minor),
            _ => Style::default().fg(theme.muted),
        };

        let checks_text = match pr.checks_pass {
            Some(true) => "✓",
            Some(false) => "✗",
            None => "—",
        };
        let checks_style = match pr.checks_pass {
            Some(true) => Style::default().fg(theme.success),
            Some(false) => Style::default().fg(theme.error),
            None => Style::default().fg(theme.muted),
        };

        let merge_text = match pr.mergeable {
            Some(true) => "✓",
            Some(false) => "✗",
            None => "—",
        };
        let merge_style = match pr.mergeable {
            Some(true) => Style::default().fg(theme.success),
            Some(false) => Style::default().fg(theme.error),
            None => Style::default().fg(theme.muted),
        };

        let lines = vec![
            Line::from(Span::styled(
                format!("#{} {}", pr.number, pr.title),
                Style::default().fg(theme.accent),
            )),
            Line::from(vec![
                Span::styled(pr.update_type.to_string(), type_style),
                Span::styled(" · ", Style::default().fg(theme.muted)),
                Span::styled(checks_text, checks_style),
                Span::styled(" · ", Style::default().fg(theme.muted)),
                Span::styled(merge_text, merge_style),
            ]),
            Line::from(Span::styled(
                format!("{} → {}", pr.branch, pr.base),
                Style::default().fg(theme.muted),
            )),
            Line::from(Span::styled(
                format!("Age: {}", pr.age_display()),
                Style::default().fg(theme.muted),
            )),
            Line::from(""),
            Line::from(Span::styled(
                "[m]erge [x]close [o]pen [r]ebase [e]recreate re[t]ry",
                Style::default().fg(theme.dim),
            )),
        ];

        Paragraph::new(lines).block(block)
    } else {
        Paragraph::new(Line::from(Span::styled(
            "No PR selected",
            Style::default().fg(theme.muted),
        )))
        .block(block)
    };

    frame.render_widget(content, area);
}
