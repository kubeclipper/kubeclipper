package tui

// KeyMap defines keyboard shortcuts for the TUI.
type KeyMap struct {
	Up       string
	Down     string
	Enter    string
	Back     string
	Follow   string
	Quit     string
	PageUp   string
	PageDown string
}

// DefaultKeyMap holds the default key bindings.
var DefaultKeyMap = KeyMap{
	Up:       "up",
	Down:     "down",
	Enter:    "enter",
	Back:     "b",
	Follow:   "f",
	Quit:     "q",
	PageUp:   "pgup",
	PageDown: "pgdown",
}
