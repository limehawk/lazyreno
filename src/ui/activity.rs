use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, List, ListItem};

use super::theme::Theme;
use crate::app::App;
use crate::types::FlashLevel;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let block = Block::default()
        .title(" Activity ")
        .borders(Borders::ALL)
        .border_style(theme.border_unfocused);

    if app.activity_log.is_empty() {
        let list = List::new(vec![ListItem::new(Line::styled(
            "No activity yet",
            Style::default().fg(theme.muted),
        ))])
        .block(block);
        frame.render_widget(list, area);
        return;
    }

    let inner_height = area.height.saturating_sub(2) as usize;
    // Show most recent entries, scrolled to bottom.
    let skip = app.activity_log.len().saturating_sub(inner_height);

    let items: Vec<ListItem> = app
        .activity_log
        .iter()
        .skip(skip)
        .map(|msg| {
            let (icon, style) = match msg.level {
                FlashLevel::Success => ("✓", Style::default().fg(theme.success)),
                FlashLevel::Error => ("✗", Style::default().fg(theme.error)),
            };
            let text = format!("{} {}", icon, msg.text);
            ListItem::new(Line::from(Span::styled(text, style)))
        })
        .collect();

    let list = List::new(items).block(block);
    frame.render_widget(list, area);
}
