use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::widgets::Paragraph;

use super::theme::Theme;
use crate::app::App;
use crate::types::FlashLevel;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    if let Some(ref flash) = app.flash {
        let style = match flash.level {
            FlashLevel::Success => Style::default().fg(theme.success),
            FlashLevel::Error => Style::default().fg(theme.error),
        };
        let paragraph = Paragraph::new(flash.text.clone()).style(style);
        frame.render_widget(paragraph, area);
    }
}
