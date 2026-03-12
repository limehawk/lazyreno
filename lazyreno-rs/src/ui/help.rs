use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::{Block, Borders, Clear, Paragraph};

use super::theme::Theme;
use crate::app::App;

pub fn render(_app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    frame.render_widget(Clear, area);

    let block = Block::default()
        .title(" Help ")
        .borders(Borders::ALL)
        .border_style(theme.border_focused);

    let accent = Style::default()
        .fg(theme.accent)
        .add_modifier(Modifier::BOLD);
    let plain = Style::default().fg(theme.text);

    let lines = vec![
        Line::from(Span::styled("Navigation", accent)),
        Line::from(Span::styled("  j/k          Move down/up", plain)),
        Line::from(Span::styled("  g/G          Jump top/bottom", plain)),
        Line::from(Span::styled("  C-u/C-d      Half page up/down", plain)),
        Line::from(Span::styled("  Tab          Next panel", plain)),
        Line::from(Span::styled("  h/l          Prev/next panel", plain)),
        Line::from(Span::styled("  Enter        Focus PR table", plain)),
        Line::from(""),
        Line::from(Span::styled("Actions", accent)),
        Line::from(Span::styled("  m            Merge PR", plain)),
        Line::from(Span::styled("  M            Merge all safe", plain)),
        Line::from(Span::styled("  c            Close PR", plain)),
        Line::from(Span::styled("  o            Open in browser", plain)),
        Line::from(Span::styled("  s            Trigger sync", plain)),
        Line::from(Span::styled("  p            Purge jobs", plain)),
        Line::from(""),
        Line::from(Span::styled("Other", accent)),
        Line::from(Span::styled("  R            Refresh", plain)),
        Line::from(Span::styled("  2            All repos", plain)),
        Line::from(Span::styled("  ?            Toggle help", plain)),
        Line::from(Span::styled("  q            Quit", plain)),
    ];

    let paragraph = Paragraph::new(lines).block(block);
    frame.render_widget(paragraph, area);
}
