package ui

import "github.com/charmbracelet/bubbles/key"

// keyMap holds the table-view bindings. The help bubble reads these to render
// the footer, so each binding carries its own help text.
type keyMap struct {
	Connect  key.Binding
	Multimon key.Binding
	New      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Cert     key.Binding
	ClearPw  key.Binding
	Help     key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Connect:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect")),
	Multimon: key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "multimon")),
	New:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Edit:     key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Delete:   key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
	Cert:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "toggle cert")),
	ClearPw:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "clear password")),
	Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:     key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
}

// ShortHelp / FullHelp satisfy help.KeyMap.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Connect, k.New, k.Edit, k.Delete, k.Cert, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Connect, k.Multimon},
		{k.New, k.Edit, k.Delete},
		{k.Cert, k.ClearPw},
		{k.Help, k.Quit},
	}
}
