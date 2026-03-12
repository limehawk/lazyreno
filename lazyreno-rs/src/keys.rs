use crossterm::event::{KeyCode, KeyEvent, KeyModifiers};

use crate::app::App;
use crate::types::{ConfirmAction, Panel};

/// Top-level key handler. Routes input based on current UI state.
pub fn handle_key(app: &mut App, key: KeyEvent) {
    // 1. Confirmation dialog intercepts all input.
    if let Some(action) = app.confirming.take() {
        if key.code == KeyCode::Char('y') {
            execute_confirmed(app, action);
        }
        // Any other key cancels — confirming is already taken/cleared.
        return;
    }

    // 2. All-repos overlay.
    if app.show_all_repos {
        handle_all_repos_key(app, key);
        return;
    }

    // 3. Help overlay.
    if app.show_help {
        match key.code {
            KeyCode::Char('?') | KeyCode::Esc => app.show_help = false,
            _ => {}
        }
        return;
    }

    // 4. Global keys.
    handle_global_key(app, key);
}

fn handle_global_key(app: &mut App, key: KeyEvent) {
    match key.code {
        KeyCode::Char('q') => {
            app.running = false;
            app.cancel_token.cancel();
        }
        KeyCode::Char('?') => {
            app.show_help = true;
        }
        KeyCode::Char('2') => {
            app.show_all_repos = true;
            app.all_repos_selected = 0;
            app.all_repos_filter.clear();
        }
        KeyCode::Char('R') => {
            // Refresh is handled in the event loop; noop here.
        }
        KeyCode::Tab => {
            app.focused_panel = app.focused_panel.next();
        }
        KeyCode::BackTab => {
            app.focused_panel = app.focused_panel.prev();
        }
        KeyCode::Char('h') | KeyCode::Left => {
            app.focused_panel = app.focused_panel.prev();
        }
        KeyCode::Char('l') | KeyCode::Right => {
            app.focused_panel = app.focused_panel.next();
        }
        KeyCode::Char('j') | KeyCode::Down => {
            app.move_selection_down();
        }
        KeyCode::Char('k') | KeyCode::Up => {
            app.move_selection_up();
        }
        KeyCode::Char('g') => {
            app.jump_top();
        }
        KeyCode::Char('G') => {
            app.jump_bottom();
        }
        KeyCode::Char('u') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            app.half_page_up(20);
        }
        KeyCode::Char('d') if key.modifiers.contains(KeyModifiers::CONTROL) => {
            app.half_page_down(20);
        }
        KeyCode::Enter => {
            if app.focused_panel == Panel::Sidebar {
                app.focused_panel = Panel::PrTable;
                app.selected_pr = 0;
            }
        }
        KeyCode::Char('s') => {
            app.dispatch_sync();
        }
        KeyCode::Char('p') => {
            app.confirming = Some(ConfirmAction::PurgeJobs);
        }

        // PR-table-specific keys.
        KeyCode::Char('m') if app.focused_panel == Panel::PrTable => {
            if let (Some(pr), Some(repo)) = (app.selected_pr(), app.selected_repo_name()) {
                app.confirming = Some(ConfirmAction::MergePr(pr.number, repo.to_string()));
            }
        }
        KeyCode::Char('M') if app.focused_panel == Panel::PrTable => {
            if let Some(repo) = app.selected_repo_name() {
                app.confirming = Some(ConfirmAction::MergeAllSafe(repo.to_string()));
            }
        }
        KeyCode::Char('c') if app.focused_panel == Panel::PrTable => {
            if let (Some(pr), Some(repo)) = (app.selected_pr(), app.selected_repo_name()) {
                app.confirming =
                    Some(ConfirmAction::ClosePr(pr.number, repo.to_string()));
            }
        }
        KeyCode::Char('o') if app.focused_panel == Panel::PrTable => {
            if let Some(pr) = app.selected_pr() {
                let _ = open::that(&pr.url);
            }
        }

        _ => {}
    }
}

fn handle_all_repos_key(app: &mut App, key: KeyEvent) {
    match key.code {
        KeyCode::Esc | KeyCode::Char('2') => {
            app.show_all_repos = false;
            app.all_repos_filter.clear();
        }
        KeyCode::Char('j') | KeyCode::Down => {
            let len = filtered_all_repos_len(app);
            if len > 0 && app.all_repos_selected < len - 1 {
                app.all_repos_selected += 1;
            }
        }
        KeyCode::Char('k') | KeyCode::Up => {
            if app.all_repos_selected > 0 {
                app.all_repos_selected -= 1;
            }
        }
        KeyCode::Backspace => {
            app.all_repos_filter.pop();
            app.all_repos_selected = 0;
        }
        KeyCode::Char(c) => {
            app.all_repos_filter.push(c);
            app.all_repos_selected = 0;
        }
        _ => {}
    }
}

/// Execute a confirmed action by dispatching the appropriate async task.
fn execute_confirmed(app: &mut App, action: ConfirmAction) {
    match action {
        ConfirmAction::MergePr(number, repo) => {
            app.dispatch_merge(number, repo);
        }
        ConfirmAction::MergeAllSafe(repo) => {
            app.dispatch_merge_all_safe(repo);
        }
        ConfirmAction::ClosePr(number, repo) => {
            if let Some(pr) = app.prs.get(&repo).and_then(|prs| {
                prs.iter().find(|p| p.number == number)
            }) {
                let branch = pr.branch.clone();
                app.dispatch_close(number, repo, branch);
            }
        }
        ConfirmAction::PurgeJobs => {
            app.dispatch_purge();
        }
    }
}

/// Count of all repos matching the current filter text.
pub fn filtered_all_repos_len(app: &App) -> usize {
    if app.all_repos_filter.is_empty() {
        return app.all_repos.len();
    }
    let lower = app.all_repos_filter.to_lowercase();
    app.all_repos
        .iter()
        .filter(|r| r.full_name.to_lowercase().contains(&lower))
        .count()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::app::App;
    use crate::types::{Repo, UpdateType, PR};
    use chrono::Utc;
    use crossterm::event::{KeyCode, KeyEvent, KeyEventKind, KeyEventState, KeyModifiers};
    use std::sync::Arc;
    use tokio::sync::mpsc;
    use tokio_util::sync::CancellationToken;

    fn make_key(code: KeyCode) -> KeyEvent {
        KeyEvent {
            code,
            modifiers: KeyModifiers::NONE,
            kind: KeyEventKind::Press,
            state: KeyEventState::NONE,
        }
    }

    fn make_app() -> App {
        let cancel = CancellationToken::new();
        let (tx, _rx) = mpsc::channel(16);
        let gh = Arc::new(
            crate::api::github::GithubClient::new("fake", "org").unwrap(),
        );
        let ren = Arc::new(crate::api::renovate::RenovateClient::new(
            "http://localhost",
            "secret",
        ));
        App::new(cancel, tx, gh, ren)
    }

    fn populate_app(app: &mut App) {
        app.repos = vec![
            Repo { full_name: "org/alpha".into(), name: "alpha".into(), pr_count: 2 },
            Repo { full_name: "org/beta".into(), name: "beta".into(), pr_count: 1 },
        ];
        app.prs.insert(
            "org/alpha".into(),
            vec![
                PR {
                    number: 1,
                    title: "PR 1".into(),
                    repo: "org/alpha".into(),
                    branch: "renovate/dep-1".into(),
                    base: "main".into(),
                    url: "https://github.com/org/alpha/pull/1".into(),
                    created_at: Utc::now(),
                    update_type: UpdateType::Minor,
                    mergeable: Some(true),
                    checks_pass: Some(true),
                },
                PR {
                    number: 2,
                    title: "PR 2".into(),
                    repo: "org/alpha".into(),
                    branch: "renovate/dep-2".into(),
                    base: "main".into(),
                    url: "https://github.com/org/alpha/pull/2".into(),
                    created_at: Utc::now(),
                    update_type: UpdateType::Major,
                    mergeable: Some(true),
                    checks_pass: Some(true),
                },
            ],
        );
        app.prs.insert(
            "org/beta".into(),
            vec![PR {
                number: 10,
                title: "PR 10".into(),
                repo: "org/beta".into(),
                branch: "renovate/dep-10".into(),
                base: "main".into(),
                url: "https://github.com/org/beta/pull/10".into(),
                created_at: Utc::now(),
                update_type: UpdateType::Patch,
                mergeable: Some(true),
                checks_pass: Some(true),
            }],
        );
    }

    #[tokio::test]
    async fn quit_sets_running_false() {
        let mut app = make_app();
        handle_key(&mut app, make_key(KeyCode::Char('q')));
        assert!(!app.running);
        assert!(app.cancel_token.is_cancelled());
    }

    #[tokio::test]
    async fn toggle_help() {
        let mut app = make_app();
        handle_key(&mut app, make_key(KeyCode::Char('?')));
        assert!(app.show_help);

        handle_key(&mut app, make_key(KeyCode::Char('?')));
        assert!(!app.show_help);
    }

    #[tokio::test]
    async fn help_blocks_other_keys() {
        let mut app = make_app();
        app.show_help = true;
        handle_key(&mut app, make_key(KeyCode::Char('q')));
        // q should NOT quit while help is showing.
        assert!(app.running);
    }

    #[tokio::test]
    async fn tab_cycles_panels() {
        let mut app = make_app();
        assert_eq!(app.focused_panel, Panel::Sidebar);

        handle_key(&mut app, make_key(KeyCode::Tab));
        assert_eq!(app.focused_panel, Panel::PrTable);

        handle_key(&mut app, make_key(KeyCode::Tab));
        assert_eq!(app.focused_panel, Panel::Detail);

        handle_key(&mut app, make_key(KeyCode::Tab));
        assert_eq!(app.focused_panel, Panel::Sidebar);
    }

    #[tokio::test]
    async fn vim_navigation_in_sidebar() {
        let mut app = make_app();
        populate_app(&mut app);

        handle_key(&mut app, make_key(KeyCode::Char('j')));
        assert_eq!(app.selected_repo, 1);

        handle_key(&mut app, make_key(KeyCode::Char('k')));
        assert_eq!(app.selected_repo, 0);

        handle_key(&mut app, make_key(KeyCode::Char('G')));
        assert_eq!(app.selected_repo, 1);

        handle_key(&mut app, make_key(KeyCode::Char('g')));
        assert_eq!(app.selected_repo, 0);
    }

    #[tokio::test]
    async fn enter_focuses_pr_table() {
        let mut app = make_app();
        populate_app(&mut app);

        handle_key(&mut app, make_key(KeyCode::Enter));
        assert_eq!(app.focused_panel, Panel::PrTable);
        assert_eq!(app.selected_pr, 0);
    }

    #[tokio::test]
    async fn confirm_dialog_y_executes() {
        let mut app = make_app();
        app.confirming = Some(ConfirmAction::PurgeJobs);
        // 'y' should clear confirming and dispatch.
        handle_key(&mut app, make_key(KeyCode::Char('y')));
        assert!(app.confirming.is_none());
    }

    #[tokio::test]
    async fn confirm_dialog_other_cancels() {
        let mut app = make_app();
        app.confirming = Some(ConfirmAction::PurgeJobs);
        handle_key(&mut app, make_key(KeyCode::Char('n')));
        assert!(app.confirming.is_none());
    }

    #[tokio::test]
    async fn all_repos_overlay_filter() {
        let mut app = make_app();
        app.all_repos = vec![
            Repo { full_name: "org/alpha".into(), name: "alpha".into(), pr_count: 0 },
            Repo { full_name: "org/beta".into(), name: "beta".into(), pr_count: 0 },
            Repo { full_name: "org/gamma".into(), name: "gamma".into(), pr_count: 0 },
        ];
        app.show_all_repos = true;

        // Type "al" to filter.
        handle_key(&mut app, make_key(KeyCode::Char('a')));
        handle_key(&mut app, make_key(KeyCode::Char('l')));
        assert_eq!(app.all_repos_filter, "al");
        assert_eq!(filtered_all_repos_len(&app), 1);

        // Backspace.
        handle_key(&mut app, make_key(KeyCode::Backspace));
        assert_eq!(app.all_repos_filter, "a");
        assert_eq!(filtered_all_repos_len(&app), 3); // all contain "a" in full_name

        // Esc closes.
        handle_key(&mut app, make_key(KeyCode::Esc));
        assert!(!app.show_all_repos);
        assert!(app.all_repos_filter.is_empty());
    }

    #[tokio::test]
    async fn merge_confirm_requires_pr_table() {
        let mut app = make_app();
        populate_app(&mut app);
        // In sidebar, 'm' should NOT set confirming.
        handle_key(&mut app, make_key(KeyCode::Char('m')));
        assert!(app.confirming.is_none());

        // Switch to PR table.
        app.focused_panel = Panel::PrTable;
        handle_key(&mut app, make_key(KeyCode::Char('m')));
        assert!(app.confirming.is_some());
    }
}
