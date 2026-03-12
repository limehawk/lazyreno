use std::time::Duration;

use crossterm::event::{self, Event as CrosstermEvent, KeyEvent};
use tokio::sync::mpsc;

use crate::types::{ActionResult, FetchResult};

/// Unified event type for the application. Merges terminal events,
/// background fetch results, action completions, and periodic ticks
/// into a single stream the main loop can select on.
pub enum AppEvent {
    Key(KeyEvent),
    #[allow(dead_code)]
    Resize(u16, u16),
    Tick,
    FetchComplete(Box<FetchResult>),
    ActionComplete(ActionResult),
}

/// Multiplexes three async event sources into one unbounded channel.
///
/// - A **real OS thread** polls crossterm (blocking I/O) for key/resize/tick
/// - Two tokio tasks forward `FetchResult` and `ActionResult` from their
///   respective mpsc channels.
pub struct EventHandler {
    rx: mpsc::UnboundedReceiver<AppEvent>,
}

impl EventHandler {
    /// Create a new event handler.
    ///
    /// * `tick_rate` — how often to emit `Tick` when no terminal event arrives
    /// * `fetch_rx` — receives `FetchResult` from the background fetcher
    /// * `action_rx` — receives `ActionResult` from dispatched user actions
    pub fn new(
        tick_rate: Duration,
        fetch_rx: mpsc::Receiver<FetchResult>,
        action_rx: mpsc::Receiver<ActionResult>,
    ) -> Self {
        let (tx, rx) = mpsc::unbounded_channel();

        // 1. Crossterm polling on a real thread (blocking I/O).
        {
            let tx = tx.clone();
            std::thread::spawn(move || {
                loop {
                    // poll() blocks for at most tick_rate, then we send Tick.
                    match event::poll(tick_rate) {
                        Ok(true) => {
                            if let Ok(evt) = event::read() {
                                let app_evt = match evt {
                                    CrosstermEvent::Key(key) => AppEvent::Key(key),
                                    CrosstermEvent::Resize(w, h) => AppEvent::Resize(w, h),
                                    _ => continue,
                                };
                                if tx.send(app_evt).is_err() {
                                    break;
                                }
                            }
                        }
                        Ok(false) => {
                            // Timeout — emit tick.
                            if tx.send(AppEvent::Tick).is_err() {
                                break;
                            }
                        }
                        Err(_) => break,
                    }
                }
            });
        }

        // 2. Forward fetch results.
        {
            let tx = tx.clone();
            tokio::spawn(async move {
                let mut fetch_rx = fetch_rx;
                while let Some(result) = fetch_rx.recv().await {
                    if tx.send(AppEvent::FetchComplete(Box::new(result))).is_err() {
                        break;
                    }
                }
            });
        }

        // 3. Forward action results.
        {
            let tx = tx.clone();
            tokio::spawn(async move {
                let mut action_rx = action_rx;
                while let Some(result) = action_rx.recv().await {
                    if tx.send(AppEvent::ActionComplete(result)).is_err() {
                        break;
                    }
                }
            });
        }

        Self { rx }
    }

    /// Await the next event from the unified channel.
    /// Returns `None` when all senders have been dropped.
    pub async fn next(&mut self) -> Option<AppEvent> {
        self.rx.recv().await
    }
}
