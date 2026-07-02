package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// form is the new/edit view: three text fields plus a cert toggle. When editing
// an existing session, origName is its current key (so a rename can drop the old
// entry); it is "" for a new session.
type form struct {
	inputs   []textinput.Model
	focus    int
	ignore   *bool // mirrors Session.IgnoreCert; cycled unset → yes → no
	origName string
}

const (
	fName = iota
	fUser
	fHost
)

func newForm() form {
	f := form{inputs: make([]textinput.Model, 3)}
	labels := []string{"name", "user", "host"}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Placeholder = labels[i]
		ti.CharLimit = 128
		f.inputs[i] = ti
	}
	f.inputs[fName].Focus()
	return f
}

func editForm(name string, user, host string, ignore *bool) form {
	f := newForm()
	f.origName = name
	f.inputs[fName].SetValue(name)
	f.inputs[fUser].SetValue(user)
	f.inputs[fHost].SetValue(host)
	f.ignore = ignore
	return f
}

func (f *form) focusField(i int) {
	for j := range f.inputs {
		if j == i {
			f.inputs[j].Focus()
		} else {
			f.inputs[j].Blur()
		}
	}
	f.focus = i
}

// update handles field navigation and typing. Returns (done, submitted): done
// when the user leaves the form, submitted true only on a valid save (enter on
// the last field with all fields filled).
func (f *form) update(msg tea.KeyMsg) (cmd tea.Cmd, done, submitted bool) {
	switch msg.String() {
	case "esc":
		return nil, true, false
	case "tab", "down":
		f.focusField((f.focus + 1) % len(f.inputs))
		return nil, false, false
	case "shift+tab", "up":
		f.focusField((f.focus - 1 + len(f.inputs)) % len(f.inputs))
		return nil, false, false
	case "ctrl+t":
		f.ignore = cycleCert(f.ignore)
		return nil, false, false
	case "enter":
		if f.complete() {
			return nil, true, true
		}
		f.focusField((f.focus + 1) % len(f.inputs))
		return nil, false, false
	}
	var c tea.Cmd
	f.inputs[f.focus], c = f.inputs[f.focus].Update(msg)
	return c, false, false
}

func (f *form) complete() bool {
	for _, in := range f.inputs {
		if in.Value() == "" {
			return false
		}
	}
	return true
}

func (f form) name() string { return f.inputs[fName].Value() }
func (f form) user() string { return f.inputs[fUser].Value() }
func (f form) host() string { return f.inputs[fHost].Value() }

func (f form) view() string {
	title := "New session"
	if f.origName != "" {
		title = "Edit session"
	}
	rows := titleStyle.Render(title) + "\n\n"
	for _, in := range f.inputs {
		rows += formLabelStyle.Render(padLabel(in.Placeholder)) + in.View() + "\n"
	}
	rows += formLabelStyle.Render(padLabel("cert")) + certLabel(f.ignore) + "\n\n"
	rows += mutedStyle.Render("tab/↑↓ move · ctrl+t toggle cert · enter save · esc cancel")
	return boxStyle.Render(rows)
}

func padLabel(s string) string {
	for len(s) < 6 {
		s += " "
	}
	return s + " "
}
