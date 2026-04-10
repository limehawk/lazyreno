use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{
    Block, Borders, List, ListItem, ListState, Scrollbar, ScrollbarOrientation, ScrollbarState,
};

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

    let inner_height = area.height.saturating_sub(2) as usize;
    let total = app.repos.len();

    let version = env!("CARGO_PKG_VERSION");
    let title = Line::from(Span::styled(
        format!(" Repos ({total}) "),
        Style::default().fg(theme.text),
    ));

    let app_name = Line::from(Span::styled(
        format!(" lazyreno v{version} "),
        Style::default().fg(theme.accent),
    ));

    let block = Block::default()
        .title(title)
        .title_bottom(app_name)
        .borders(Borders::ALL)
        .border_style(border_style);

    let items: Vec<ListItem> = app
        .repos
        .iter()
        .map(|repo| {
            let text = format!("  {} ({})", repo.name, repo.pr_count);
            ListItem::new(Line::styled(text, Style::default().fg(theme.text)))
        })
        .collect();

    let highlight_style = if focused {
        Style::default()
            .fg(theme.accent)
            .add_modifier(Modifier::BOLD)
    } else {
        Style::default().fg(theme.accent)
    };

    let list = List::new(items)
        .block(block)
        .highlight_symbol("▸ ")
        .highlight_style(highlight_style);

    let selected = if total > 0 {
        Some(app.selected_repo)
    } else {
        None
    };
    let mut list_state = ListState::default().with_selected(selected);
    frame.render_stateful_widget(list, area, &mut list_state);

    // Scrollbar — only when repos overflow the visible area.
    if total > inner_height {
        let mut scrollbar_state = ScrollbarState::new(total.saturating_sub(inner_height))
            .position(list_state.offset());
        let scrollbar = Scrollbar::new(ScrollbarOrientation::VerticalRight)
            .style(Style::default().fg(theme.muted))
            .begin_symbol(None)
            .end_symbol(None);
        let scrollbar_area = Rect {
            x: area.x,
            y: area.y + 1,
            width: area.width,
            height: area.height.saturating_sub(2),
        };
        frame.render_stateful_widget(scrollbar, scrollbar_area, &mut scrollbar_state);
    }
}
