use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::Line;
use ratatui::widgets::{Block, Borders, List, ListItem};
use ratatui::Frame;

use crate::app::App;
use crate::types::Panel;
use super::theme::Theme;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let focused = app.focused_panel == Panel::Sidebar;
    let border_style = if focused {
        theme.border_focused
    } else {
        theme.border_unfocused
    };

    let block = Block::default()
        .title(" Repos ")
        .borders(Borders::ALL)
        .border_style(border_style);

    let items: Vec<ListItem> = app
        .repos
        .iter()
        .enumerate()
        .map(|(i, repo)| {
            let selected = i == app.selected_repo;
            let prefix = if selected { "▸" } else { " " };
            let text = format!("{} {} ({})", prefix, repo.name, repo.pr_count);
            let style = if selected && focused {
                Style::default()
                    .fg(theme.accent)
                    .add_modifier(Modifier::BOLD)
            } else if selected {
                Style::default().fg(theme.accent)
            } else {
                Style::default().fg(theme.text)
            };
            ListItem::new(Line::styled(text, style))
        })
        .collect();

    let list = List::new(items).block(block);
    frame.render_widget(list, area);
}
