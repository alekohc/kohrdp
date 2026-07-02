package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"rdpkoh/internal/config"
)

// form is the new/edit view: text fields plus a cert toggle. When editing an
// existing session, origName is its current key (so a rename can drop the old
// entry); it is "" for a new session. Only name/user/host are required.
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
	fDomain
	fGateway
	fSize
	fDrives
)

func newForm() form {
	labels := []string{"name", "user", "host", "domain", "gateway", "size", "drives"}
	placeholders := []string{
		"name", "user", "host",
		"domain (optional)", "gateway (optional)", "WxH (optional)",
		"name,path name,path … (optional)",
	}
	f := form{inputs: make([]textinput.Model, len(labels))}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 256
		f.inputs[i] = ti
	}
	f.inputs[fName].Focus()
	return f
}

func editForm(name string, s config.Session) form {
	f := newForm()
	f.origName = name
	f.inputs[fName].SetValue(name)
	f.inputs[fUser].SetValue(s.User)
	f.inputs[fHost].SetValue(s.Host)
	f.inputs[fDomain].SetValue(s.Domain)
	f.inputs[fGateway].SetValue(s.Gateway)
	f.inputs[fSize].SetValue(s.Size)
	f.inputs[fDrives].SetValue(strings.Join(s.Drives, " "))
	f.ignore = s.IgnoreCert
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
// when the user leaves the form, submitted true only on a valid save (enter with
// name/user/host filled).
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
	return f.inputs[fName].Value() != "" &&
		f.inputs[fUser].Value() != "" &&
		f.inputs[fHost].Value() != ""
}

func (f form) name() string    { return f.inputs[fName].Value() }
func (f form) user() string    { return f.inputs[fUser].Value() }
func (f form) host() string    { return f.inputs[fHost].Value() }
func (f form) domain() string  { return f.inputs[fDomain].Value() }
func (f form) gateway() string { return f.inputs[fGateway].Value() }
func (f form) size() string    { return f.inputs[fSize].Value() }
func (f form) drives() []string {
	return strings.Fields(f.inputs[fDrives].Value())
}

func (f form) view() string {
	title := "New session"
	if f.origName != "" {
		title = "Edit session"
	}
	labels := []string{"name", "user", "host", "domain", "gateway", "size", "drives"}
	rows := titleStyle.Render(title) + "\n\n"
	for i, in := range f.inputs {
		rows += formLabelStyle.Render(padLabel(labels[i])) + in.View() + "\n"
	}
	rows += formLabelStyle.Render(padLabel("cert")) + certLabel(f.ignore) + "\n\n"
	rows += mutedStyle.Render("tab/↑↓ move · ctrl+t toggle cert · enter save · esc cancel")
	return boxStyle.Render(rows)
}

func padLabel(s string) string {
	for len(s) < 7 {
		s += " "
	}
	return s + " "
}
