use ratatui::Frame;
use ratatui::layout::{Constraint, Rect};
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Cell, Row, Table};

use super::theme::Theme;
use crate::app::App;
use crate::types::{Panel, UpdateType};

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let focused = app.focused_panel == Panel::PrTable;
    let border_style = if focused {
        theme.border_focused
    } else {
        theme.border_unfocused
    };

    let repo_name = app.selected_repo_name().unwrap_or("—");
    let prs = app.current_prs();

    // Inner height minus 2 for borders, minus 1 for header row.
    let inner_height = area.height.saturating_sub(3) as usize;
    let total = prs.len();
    let scroll_arrow = if total > inner_height {
        if app.selected_pr == 0 {
            " ↓"
        } else if app.selected_pr >= total - 1 {
            " ↑"
        } else {
            " ↕"
        }
    } else {
        ""
    };

    let title = Line::from(vec![
        Span::raw(format!(" PRs — {} ({}) ", repo_name, total)),
        Span::styled(scroll_arrow, Style::default().fg(theme.muted)),
    ]);

    let block = Block::default()
        .title(title)
        .borders(Borders::ALL)
        .border_style(border_style);

    let header_style = Style::default().fg(theme.muted);
    let header = Row::new(vec![
        Cell::from(Span::styled("#", header_style)),
        Cell::from(Span::styled("Title", header_style)),
        Cell::from(Span::styled("Type", header_style)),
        Cell::from(Span::styled("Checks", header_style)),
        Cell::from(Span::styled("Merge", header_style)),
        Cell::from(Span::styled("Age", header_style)),
    ]);

    let rows: Vec<Row> = prs
        .iter()
        .enumerate()
        .map(|(i, pr)| {
            let selected = i == app.selected_pr;

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

            let row = Row::new(vec![
                Cell::from(format!("{}", pr.number)),
                Cell::from(pr.title.clone()),
                Cell::from(Span::styled(pr.update_type.to_string(), type_style)),
                Cell::from(Span::styled(checks_text, checks_style)),
                Cell::from(Span::styled(merge_text, merge_style)),
                Cell::from(Span::styled(
                    pr.age_display(),
                    Style::default().fg(theme.muted),
                )),
            ]);

            if selected {
                row.style(
                    Style::default()
                        .fg(theme.accent)
                        .add_modifier(Modifier::BOLD),
                )
            } else {
                row
            }
        })
        .collect();

    let widths = [
        Constraint::Length(6),
        Constraint::Min(20),
        Constraint::Length(7),
        Constraint::Length(6),
        Constraint::Length(6),
        Constraint::Length(5),
    ];

    let table = Table::new(rows, widths).header(header).block(block);

    frame.render_widget(table, area);
}
