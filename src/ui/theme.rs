use ratatui::style::{Color, Style};

#[allow(dead_code)]
pub struct Theme {
    pub accent: Color,
    pub dim: Color,
    pub text: Color,
    pub muted: Color,
    pub success: Color,
    pub error: Color,
    pub warning: Color,
    pub major: Color,
    pub minor: Color,
    pub patch: Color,
    pub badge_bg: Color,
    pub border_focused: Style,
    pub border_unfocused: Style,
}

impl Theme {
    pub fn new(accent_name: &str) -> Self {
        let accent = parse_color(accent_name);
        Self {
            accent,
            dim: Color::DarkGray,
            text: Color::White,
            muted: Color::Gray,
            success: Color::Green,
            error: Color::Red,
            warning: Color::Yellow,
            major: Color::Red,
            minor: Color::Green,
            patch: Color::Green,
            badge_bg: Color::Rgb(28, 35, 51),
            border_focused: Style::default().fg(accent),
            border_unfocused: Style::default().fg(Color::DarkGray),
        }
    }
}

fn parse_color(name: &str) -> Color {
    match name.to_lowercase().as_str() {
        "cyan" => Color::Cyan,
        "magenta" => Color::Magenta,
        "blue" => Color::Blue,
        "green" => Color::Green,
        "yellow" => Color::Yellow,
        "red" => Color::Red,
        "white" => Color::White,
        _ => Color::Cyan,
    }
}
