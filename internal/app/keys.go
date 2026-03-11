package app

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Quit      key.Binding
	Help      key.Binding
	Refresh   key.Binding
	Repos     key.Binding // toggle repos overlay
	FocusNext key.Binding
	FocusPrev key.Binding
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Top       key.Binding
	Bottom    key.Binding
	Enter     key.Binding
	Filter    key.Binding
	Escape    key.Binding
	HalfDown  key.Binding
	HalfUp    key.Binding

	// Action bindings
	Merge     key.Binding
	MergeSafe key.Binding
	Close     key.Binding
	Open      key.Binding
	Sync      key.Binding
	Purge     key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Merge, k.MergeSafe, k.Close, k.Open, k.Sync, k.Purge, k.Repos, k.Help}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Top, k.Bottom, k.HalfUp, k.HalfDown},
		{k.FocusNext, k.FocusPrev, k.Enter, k.Filter, k.Escape},
		{k.Merge, k.MergeSafe, k.Close, k.Open, k.Sync, k.Purge},
		{k.Repos, k.Refresh, k.Help, k.Quit},
	}
}

var GlobalKeys = KeyMap{
	Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Refresh:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
	Repos:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "repos")),
	FocusNext: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
	FocusPrev: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("S-tab", "prev panel")),
	Up:        key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
	Down:      key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
	Left:      key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "left")),
	Right:     key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "right")),
	Top:       key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
	Bottom:    key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
	Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Escape:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	HalfDown:  key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("C-d", "half page down")),
	HalfUp:    key.NewBinding(key.WithKeys("ctrl+u"), key.WithHelp("C-u", "half page up")),

	Merge:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merge PR")),
	MergeSafe: key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "safe merge")),
	Close:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "close PR")),
	Open:      key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "browser")),
	Sync:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sync")),
	Purge:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "purge")),
}
