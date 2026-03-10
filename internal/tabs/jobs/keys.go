package jobs

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Retry key.Binding
	Purge key.Binding
}

var Keys = KeyMap{
	Retry: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
	Purge: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "purge failed")),
}
