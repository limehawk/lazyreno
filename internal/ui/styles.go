package ui

import (
	"bufio"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// Theme colors — loaded from btop.theme or ANSI fallback.
var (
	Accent       color.Color
	Title        color.Color
	Dim          lipgloss.Style
	Text         color.Color
	SelectedBG   color.Color
	SelectedFG   color.Color
	Border       color.Color
	SecondAccent color.Color
)

// Pre-built styles — initialized after colors load.
var (
	Bold        lipgloss.Style
	DimStyle    lipgloss.Style
	AccentText  lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
	SuccessText lipgloss.Style
	ActiveBorder   lipgloss.Style
	InactiveBorder lipgloss.Style
	ShortcutKey    lipgloss.Style
)

func init() {
	theme := loadBtopTheme()
	if theme != nil {
		Accent = lipgloss.Color(theme["hi_fg"])
		Title = lipgloss.Color(theme["title"])
		Text = lipgloss.Color(theme["main_fg"])
		SelectedBG = lipgloss.Color(theme["selected_bg"])
		SelectedFG = lipgloss.Color(theme["selected_fg"])
		Border = lipgloss.Color(theme["div_line"])
		SecondAccent = lipgloss.Color(theme["proc_misc"])

		dimColor := lipgloss.Color(theme["inactive_fg"])
		Dim = lipgloss.NewStyle().Foreground(dimColor)
	} else {
		// ANSI fallback.
		Accent = lipgloss.Color("4")
		Title = lipgloss.Color("7")
		Text = lipgloss.Color("7")
		SelectedBG = lipgloss.Color("8")
		SelectedFG = lipgloss.Color("15")
		Border = lipgloss.Color("8")
		SecondAccent = lipgloss.Color("6")

		Dim = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	}

	Bold = lipgloss.NewStyle().Bold(true)
	DimStyle = Dim
	AccentText = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	ErrorText = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	WarningText = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	SuccessText = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	ActiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Accent).
		Padding(0, 1)

	InactiveBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Border).
		Padding(0, 1)

	ShortcutKey = lipgloss.NewStyle().Foreground(SecondAccent).Bold(true)
}

// HuhTheme returns a huh form theme using the btop theme colors.
func HuhTheme(isDark bool) *huh.Styles {
	t := huh.ThemeBase(isDark)

	t.Focused.Base = t.Focused.Base.BorderForeground(Border)
	t.Focused.Title = t.Focused.Title.Foreground(Accent).Bold(true)
	t.Focused.Description = Dim
	t.Focused.FocusedButton = lipgloss.NewStyle().
		Padding(0, 2).MarginRight(1).
		Foreground(SelectedFG).Background(Accent).Bold(true)
	t.Focused.BlurredButton = lipgloss.NewStyle().
		Padding(0, 2).MarginRight(1).
		Foreground(Text).Background(SelectedBG)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())

	return t
}

// PanelBorder returns the border style for a panel.
func PanelBorder(focused bool) lipgloss.Style {
	if focused {
		return ActiveBorder
	}
	return InactiveBorder
}

// PanelTitle returns a styled title string.
func PanelTitle(title string, focused bool) string {
	if focused {
		return lipgloss.NewStyle().Foreground(Title).Bold(true).Render(title)
	}
	return Dim.Render(title)
}

// PRAgeForeground returns a color based on PR age.
// Fresh = green, aging = yellow, stale = red.
func PRAgeForeground(created time.Time) color.Color {
	d := time.Since(created)
	switch {
	case d < 24*time.Hour:
		return lipgloss.Color("2") // green
	case d < 3*24*time.Hour:
		return lipgloss.Color("3") // yellow
	default:
		return lipgloss.Color("1") // red
	}
}

// loadBtopTheme parses ~/.config/omarchy/current/theme/btop.theme.
// Returns nil if the file doesn't exist or can't be read.
func loadBtopTheme() map[string]string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	path := filepath.Join(home, ".config", "omarchy", "current", "theme", "btop.theme")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	theme := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: theme[key]="value"
		if !strings.HasPrefix(line, "theme[") {
			continue
		}
		line = strings.TrimPrefix(line, "theme[")
		idx := strings.Index(line, "]=")
		if idx < 0 {
			continue
		}
		key := line[:idx]
		val := strings.Trim(line[idx+2:], "\"")
		theme[key] = val
	}

	// Require at minimum the key fields we use.
	required := []string{"hi_fg", "main_fg", "inactive_fg", "div_line"}
	for _, k := range required {
		if theme[k] == "" {
			return nil
		}
	}

	return theme
}
