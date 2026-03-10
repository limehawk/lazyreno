package status

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Sync  key.Binding
	Purge key.Binding
}

var Keys = KeyMap{
	Sync:  key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sync now")),
	Purge: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "purge failed")),
}
