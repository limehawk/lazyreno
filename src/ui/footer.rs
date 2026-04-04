use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::Paragraph;

use super::theme::Theme;
use crate::app::App;
use crate::types::Panel;

/// Shortcut hints at the bottom. All keys shown always; inactive ones are dimmed.
pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let on_pr = app.focused_panel == Panel::PrTable;
    let on_pr_or_sidebar = on_pr || app.focused_panel == Panel::Sidebar;

    let key_active = Style::default()
        .fg(theme.accent)
        .add_modifier(Modifier::BOLD);
    let desc_active = Style::default().fg(theme.muted);
    let key_dim = Style::default().fg(theme.dim);
    let desc_dim = Style::default().fg(theme.dim);
    let sep = Style::default().fg(theme.dim);

    // (key, description, active?)
    let hints: &[(&str, &str, bool)] = &[
        ("j/k", "navigate", true),
        ("m", "merge", on_pr),
        ("M", "merge safe", on_pr),
        ("A", "merge all", on_pr_or_sidebar),
        ("x", "close", on_pr),
        ("o", "browser", on_pr),
        ("s", "sync", true),
        ("P", "purge", true),
        ("Tab", "panel", true),
        ("a", "repos", true),
        ("?", "help", true),
        ("q", "quit", true),
    ];

    let mut spans = Vec::new();
    for (i, (key, desc, active)) in hints.iter().enumerate() {
        if i > 0 {
            spans.push(Span::styled(" │ ", sep));
        }
        let (ks, ds) = if *active {
            (key_active, desc_active)
        } else {
            (key_dim, desc_dim)
        };
        spans.push(Span::styled(*key, ks));
        spans.push(Span::styled(format!(" {}", desc), ds));
    }

    let line = Line::from(spans);
    let paragraph = Paragraph::new(line);
    frame.render_widget(paragraph, area);
}
