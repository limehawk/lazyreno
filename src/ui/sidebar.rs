use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, List, ListItem};

use super::theme::Theme;
use crate::app::App;
use crate::types::Panel;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let focused = app.focused_panel == Panel::Sidebar;
    let border_style = if focused {
        theme.border_focused
    } else {
        theme.border_unfocused
    };

    // Inner height = area minus 2 for borders.
    let inner_height = area.height.saturating_sub(2) as usize;
    let total = app.repos.len();
    let scroll_indicator = scroll_hint(app.selected_repo, total, inner_height, theme);

    let title = match scroll_indicator {
        Some(ref hint) => Line::from(vec![
            Span::raw(" Repos "),
            hint.clone(),
        ]),
        None => Line::from(" Repos "),
    };

    let block = Block::default()
        .title(title)
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

/// Returns a scroll hint span if the list overflows the visible area.
/// Shows ↑/↓/↕ depending on scroll position.
fn scroll_hint(selected: usize, total: usize, visible: usize, theme: &Theme) -> Option<Span<'static>> {
    if total <= visible {
        return None;
    }
    let arrow = if selected == 0 {
        "↓"
    } else if selected >= total - 1 {
        "↑"
    } else {
        "↕"
    };
    Some(Span::styled(
        arrow.to_string(),
        Style::default().fg(theme.muted),
    ))
}
