use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Clear, Paragraph};

use super::theme::Theme;
use crate::app::App;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    frame.render_widget(Clear, area);

    let block = Block::default()
        .title(" Confirm ")
        .borders(Borders::ALL)
        .border_style(Style::default().fg(theme.warning));

    let lines = if let Some(ref action) = app.confirming {
        vec![
            Line::from(Span::styled(
                action.to_string(),
                Style::default().fg(theme.warning),
            )),
            Line::from(""),
            Line::from(Span::styled(
                "[y/Enter] yes  [n/Esc] no",
                Style::default().fg(theme.muted),
            )),
        ]
    } else {
        vec![]
    };

    let paragraph = Paragraph::new(lines).block(block);
    frame.render_widget(paragraph, area);
}
