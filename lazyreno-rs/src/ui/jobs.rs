use ratatui::Frame;
use ratatui::layout::Rect;
use ratatui::style::Style;
use ratatui::text::Line;
use ratatui::widgets::{Block, Borders, List, ListItem};

use super::theme::Theme;
use crate::app::App;
use crate::types::JobState;

pub fn render(app: &App, frame: &mut Frame, area: Rect, theme: &Theme) {
    let block = Block::default()
        .title(" Jobs ")
        .borders(Borders::ALL)
        .border_style(theme.border_unfocused);

    if app.jobs.is_empty() {
        // Try to show last finished from system_status.
        let content = if let Some(ref st) = app.system_status {
            if let Some(ref job) = st.last_finished {
                let dur = job
                    .duration
                    .map(|d| format!("{}s", d.as_secs()))
                    .unwrap_or_default();
                let text = format!("Last: {} ({})", job.repo, dur);
                vec![ListItem::new(Line::styled(
                    text,
                    Style::default().fg(theme.muted),
                ))]
            } else {
                vec![ListItem::new(Line::styled(
                    "No jobs",
                    Style::default().fg(theme.muted),
                ))]
            }
        } else {
            vec![ListItem::new(Line::styled(
                "No jobs",
                Style::default().fg(theme.muted),
            ))]
        };

        let list = List::new(content).block(block);
        frame.render_widget(list, area);
        return;
    }

    let items: Vec<ListItem> = app
        .jobs
        .iter()
        .map(|job| {
            let (icon, style, suffix) = match job.state {
                JobState::Running => {
                    let dur = job
                        .duration
                        .map(|d| format!(" ({}s)", d.as_secs()))
                        .unwrap_or_default();
                    ("⟳", Style::default().fg(theme.warning), dur)
                }
                JobState::Pending => ("⏳", Style::default().fg(theme.muted), String::new()),
                JobState::Failed => ("✗", Style::default().fg(theme.error), String::new()),
                JobState::Finished => {
                    let dur = job
                        .duration
                        .map(|d| format!(" ({}s)", d.as_secs()))
                        .unwrap_or_default();
                    ("✓", Style::default().fg(theme.success), dur)
                }
            };
            let text = format!("{} {}{}", icon, job.repo, suffix);
            ListItem::new(Line::styled(text, style))
        })
        .collect();

    let list = List::new(items).block(block);
    frame.render_widget(list, area);
}
