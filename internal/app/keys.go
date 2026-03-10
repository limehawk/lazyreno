package app

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Quit      key.Binding
	Help      key.Binding
	Refresh   key.Binding
	Tab1      key.Binding
	Tab2      key.Binding
	Tab3      key.Binding
	Tab4      key.Binding
	NextTab   key.Binding
	PrevTab   key.Binding
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
}

var GlobalKeys = KeyMap{
	Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Refresh:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "refresh")),
	Tab1:      key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "PRs")),
	Tab2:      key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "Repos")),
	Tab3:      key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "Jobs")),
	Tab4:      key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "Status")),
	NextTab:   key.NewBinding(key.WithKeys("]"), key.WithHelp("]", "next tab")),
	PrevTab:   key.NewBinding(key.WithKeys("["), key.WithHelp("[", "prev tab")),
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
}
