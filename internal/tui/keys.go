package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all key bindings
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	Enter    key.Binding
	Add      key.Binding
	Done     key.Binding
	Delete   key.Binding
	Project  key.Binding
	Priority key.Binding
	Help     key.Binding
	Quit     key.Binding
	Escape   key.Binding
	Logout   key.Binding
	Refresh  key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:    key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "left pane")),
	Right:   key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "right pane")),
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select/toggle")),
	Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add task")),
	Done:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "toggle done")),
	Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Project: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "new project")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Escape:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Logout:  key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "logout")),
	Refresh: key.NewBinding(key.WithKeys("R", "r"), key.WithHelp("R", "refresh/sync")),
}
