use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::Line;
use ratatui::widgets::{Block, Borders, Clear, List, ListItem};

use super::theme::Theme;
use crate::app::App;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    frame.render_widget(Clear, area);

    let filter = app.all_repos_filter.to_lowercase();
    let filtered: Vec<(usize, &crate::types::Repo)> = app
        .all_repos
        .iter()
        .filter(|r| filter.is_empty() || r.full_name.to_lowercase().contains(&filter))
        .enumerate()
        .collect();

    let count = filtered.len();
    let title = if app.all_repos_filter.is_empty() {
        format!(" All Repos ({}) ", count)
    } else {
        format!(" All Repos — filter: {} ({}) ", app.all_repos_filter, count)
    };

    let block = Block::default()
        .title(title)
        .borders(Borders::ALL)
        .border_style(theme.border_focused);

    let items: Vec<ListItem> = filtered
        .iter()
        .map(|(i, repo)| {
            let pr_count = app.prs.get(&repo.full_name).map(|v| v.len()).unwrap_or(0);
            let text = format!("{} ({} PRs)", repo.full_name, pr_count);
            let style = if *i == app.all_repos_selected {
                Style::default()
                    .fg(theme.accent)
                    .add_modifier(Modifier::BOLD)
            } else {
                Style::default().fg(theme.text)
            };
            ListItem::new(Line::styled(text, style))
        })
        .collect();

    let list = List::new(items).block(block);
    frame.render_widget(list, area);
}
