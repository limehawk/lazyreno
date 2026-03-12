use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Paragraph};

use super::theme::Theme;
use crate::app::App;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let block = Block::default()
        .title(" System ")
        .borders(Borders::ALL)
        .border_style(theme.border_unfocused);

    let content = if let Some(ref st) = app.system_status {
        let lines = vec![
            Line::from(vec![
                Span::styled("● ", Style::default().fg(theme.success)),
                Span::styled(format!("v{}", st.version), Style::default().fg(theme.text)),
            ]),
            Line::from(Span::styled(
                format!("Up {}", st.uptime),
                Style::default().fg(theme.muted),
            )),
            Line::from(Span::styled(
                format!(
                    "Queue: {} · Running: {} · Failed: {}",
                    st.queue_size, st.running_jobs, st.failed_jobs
                ),
                Style::default().fg(theme.muted),
            )),
        ];
        Paragraph::new(lines).block(block)
    } else {
        Paragraph::new(Line::from(Span::styled(
            "No status",
            Style::default().fg(theme.muted),
        )))
        .block(block)
    };

    frame.render_widget(content, area);
}
