package keys

import "charm.land/bubbles/v2/key"

// KeyMap holds key bindings for general navigation (vi-style and arrows).
type KeyMap struct {
	MoveUp     key.Binding
	MoveDown   key.Binding
	MoveLeft   key.Binding
	MoveRight  key.Binding
	GotoTop    key.Binding
	GotoBottom key.Binding
}

// DefaultKeyMap returns the default navigation keymap.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		MoveUp:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "move up")),
		MoveDown:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "move down")),
		MoveLeft:   key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("left/h", "move left")),
		MoveRight:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("right/l", "move right")),
		GotoTop:    key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g/home", "go to top")),
		GotoBottom: key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G/end", "go to bottom")),
	}
}

// Default is the shared default navigation keymap used by components.
var Default = DefaultKeyMap()
