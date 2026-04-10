use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::Line;
use ratatui::widgets::{
    Block, Borders, Clear, List, ListItem, ListState, Scrollbar, ScrollbarOrientation,
    ScrollbarState,
};

use super::theme::Theme;
use crate::app::App;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    frame.render_widget(Clear, area);

    let filter = app.all_repos_filter.to_lowercase();
    let visible = app.visible_all_repos();
    let filtered: Vec<&crate::types::Repo> = visible
        .into_iter()
        .filter(|r| filter.is_empty() || r.full_name.to_lowercase().contains(&filter))
        .collect();

    let count = filtered.len();
    let title = if app.all_repos_filter.is_empty() {
        format!(" All Repos ({}) ", count)
    } else {
        format!(" All Repos — filter: {} ({}) ", app.all_repos_filter, count)
    };

    let block = Block::default()
        .title(title)
        .title_bottom(Line::styled(" Esc close  type to filter ", Style::default().fg(theme.muted)))
        .borders(Borders::ALL)
        .border_style(theme.border_focused);

    let items: Vec<ListItem> = filtered
        .iter()
        .map(|repo| {
            let pr_count = app.prs.get(&repo.full_name).map(|v| v.len()).unwrap_or(0);
            let text = format!("{} ({} PRs)", repo.full_name, pr_count);
            ListItem::new(Line::styled(text, Style::default().fg(theme.text)))
        })
        .collect();

    let list = List::new(items)
        .block(block)
        .highlight_style(
            Style::default()
                .fg(theme.accent)
                .add_modifier(Modifier::BOLD),
        );

    let selected = if count > 0 {
        Some(app.all_repos_selected.min(count - 1))
    } else {
        None
    };
    let mut list_state = ListState::default().with_selected(selected);
    frame.render_stateful_widget(list, area, &mut list_state);

    // Scrollbar — only when the list overflows.
    let inner_height = area.height.saturating_sub(2) as usize;
    if count > inner_height {
        let mut scrollbar_state = ScrollbarState::new(count.saturating_sub(inner_height))
            .position(list_state.offset());
        let scrollbar = Scrollbar::new(ScrollbarOrientation::VerticalRight)
            .style(Style::default().fg(theme.muted))
            .begin_symbol(None)
            .end_symbol(None);
        // Render inside the border (inset by 1 on each side).
        let scrollbar_area = Rect {
            x: area.x,
            y: area.y + 1,
            width: area.width,
            height: area.height.saturating_sub(2),
        };
        frame.render_stateful_widget(scrollbar, scrollbar_area, &mut scrollbar_state);
    }
}
