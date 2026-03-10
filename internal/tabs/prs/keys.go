package prs

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Merge     key.Binding
	MergeSafe key.Binding
	Close     key.Binding
	Open      key.Binding
}

var Keys = KeyMap{
	Merge:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merge")),
	MergeSafe: key.NewBinding(key.WithKeys("M"), key.WithHelp("M", "merge safe")),
	Close:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "close")),
	Open:      key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open in browser")),
}
