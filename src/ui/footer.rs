use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::{Modifier, Style};
use ratatui::text::{Line, Span};
use ratatui::widgets::Paragraph;

use super::theme::Theme;
use crate::app::App;
use crate::types::Panel;

/// Context-sensitive shortcut hints with badge-style keys on two lines.
/// Line 1: action keys (change based on focused panel).
/// Line 2: navigation and global keys (always shown).
pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let on_pr = app.focused_panel == Panel::PrTable;
    let on_pr_or_sidebar = on_pr || app.focused_panel == Panel::Sidebar;

    let badge_key = Style::default()
        .fg(theme.accent)
        .bg(theme.badge_bg)
        .add_modifier(Modifier::BOLD);
    let desc_style = Style::default().fg(theme.muted);

    // Line 1: action keys (context-sensitive)
    let actions: &[(&str, &str, bool)] = &[
        ("m", "merge", on_pr),
        ("M", "safe", on_pr),
        ("A", "all", on_pr_or_sidebar),
        ("x", "close", on_pr),
        ("r", "rebase", on_pr),
        ("R", "reb-all", on_pr_or_sidebar),
        ("e", "recreate", on_pr),
        ("t", "retry", on_pr),
        ("o", "open", on_pr),
        ("s", "sync", true),
        ("P", "purge", true),
    ];

    // Line 2: navigation and global keys (always shown)
    let nav: &[(&str, &str)] = &[
        ("j/k", "nav"),
        ("h/l", "panel"),
        ("g/G", "top/bot"),
        ("Tab", "cycle"),
        ("a", "repos"),
        ("f", "forks"),
        ("?", "help"),
        ("q", "quit"),
    ];

    let line1 = build_active_line(actions, badge_key, desc_style);
    let line2 = build_always_line(nav, badge_key, desc_style);

    let paragraph = Paragraph::new(vec![line1, line2]);
    frame.render_widget(paragraph, area);
}

fn build_active_line<'a>(hints: &[(&'a str, &'a str, bool)], badge: Style, desc: Style) -> Line<'a> {
    let mut spans = Vec::new();
    for (key, label, active) in hints {
        if !active {
            continue;
        }
        if !spans.is_empty() {
            spans.push(Span::raw(" "));
        }
        spans.push(Span::styled(format!(" {} ", key), badge));
        spans.push(Span::styled(format!("{} ", label), desc));
    }
    Line::from(spans)
}

fn build_always_line<'a>(hints: &[(&'a str, &'a str)], badge: Style, desc: Style) -> Line<'a> {
    let mut spans = Vec::new();
    for (key, label) in hints {
        if !spans.is_empty() {
            spans.push(Span::raw(" "));
        }
        spans.push(Span::styled(format!(" {} ", key), badge));
        spans.push(Span::styled(format!("{} ", label), desc));
    }
    Line::from(spans)
}
