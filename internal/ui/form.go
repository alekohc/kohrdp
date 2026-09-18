package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"kohrdp/internal/config"
)

// form is the new/edit view: text fields, a cert toggle, and redirection
// toggles, all in one focus cycle. When editing an existing session, origName is
// its current key (so a rename can drop the old entry); "" for a new session.
// Only name/user/host are required.
type form struct {
	inputs   []textinput.Model
	ignore   *bool  // cert; cycled unset → yes → no
	redir    []bool // parallel to redirLabels
	focus    int    // 0..len(inputs)-1 = inputs, then cert, then redir
	origName string
}

const (
	fName = iota
	fUser
	fHost
	fDomain
	fGateway
	fSize
	fScale
	fDrives
)

const (
	rClipboard = iota
	rSound
	rMicrophone
	rPrinter
	rSmartcard
	rLAN
)

var (
	inputLabels  = []string{"name", "user", "host", "domain", "gateway", "size", "scale", "drives"}
	redirLabels  = []string{"clipboard", "sound", "microphone", "printer", "smartcard", "lan"}
	redirDefault = []bool{true, true, false, false, false, false}
)

func newForm() form {
	placeholders := []string{
		"name", "user", "host",
		"domain (optional)", "gateway (optional)", "WxH (optional)",
		"DPI %, e.g. 125 (optional)",
		"name,path name,path … (optional)",
	}
	f := form{inputs: make([]textinput.Model, len(inputLabels))}
	for i := range f.inputs {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = placeholders[i]
		ti.CharLimit = 256
		f.inputs[i] = ti
	}
	f.redir = append([]bool(nil), redirDefault...)
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
	f.inputs[fScale].SetValue(s.Scale)
	f.inputs[fDrives].SetValue(strings.Join(s.Drives, " "))
	f.ignore = s.IgnoreCert
	f.redir[rClipboard] = config.BoolOr(s.Clipboard, redirDefault[rClipboard])
	f.redir[rSound] = config.BoolOr(s.Sound, redirDefault[rSound])
	f.redir[rMicrophone] = config.BoolOr(s.Microphone, redirDefault[rMicrophone])
	f.redir[rPrinter] = config.BoolOr(s.Printer, redirDefault[rPrinter])
	f.redir[rSmartcard] = config.BoolOr(s.Smartcard, redirDefault[rSmartcard])
	f.redir[rLAN] = config.BoolOr(s.LAN, redirDefault[rLAN])
	return f
}

func (f *form) fieldCount() int { return len(f.inputs) + 1 + len(f.redir) }

func (f *form) moveFocus(delta int) {
	n := f.fieldCount()
	f.focus = (f.focus + delta + n) % n
	for j := range f.inputs {
		if j == f.focus {
			f.inputs[j].Focus()
		} else {
			f.inputs[j].Blur()
		}
	}
}

// toggleFocused flips the toggle the focus is on (cert or a redirection flag).
func (f *form) toggleFocused() {
	i := f.focus - len(f.inputs)
	if i == 0 {
		f.ignore = cycleCert(f.ignore)
		return
	}
	f.redir[i-1] = !f.redir[i-1]
}

// update handles field navigation, typing, and toggles. Returns (done,
// submitted): done when the user leaves the form, submitted true only on a valid
// save (enter with name/user/host filled).
func (f *form) update(msg tea.KeyMsg) (cmd tea.Cmd, done, submitted bool) {
	switch msg.String() {
	case "esc":
		return nil, true, false
	case "tab", "down":
		f.moveFocus(1)
		return nil, false, false
	case "shift+tab", "up":
		f.moveFocus(-1)
		return nil, false, false
	case "enter":
		if f.complete() {
			return nil, true, true
		}
		f.moveFocus(1)
		return nil, false, false
	}
	if f.focus >= len(f.inputs) {
		switch msg.String() {
		case " ", "left", "right":
			f.toggleFocused()
		}
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
func (f form) scale() string   { return f.inputs[fScale].Value() }
func (f form) drives() []string {
	return strings.Fields(f.inputs[fDrives].Value())
}

func (f form) view() string {
	title := "New session"
	if f.origName != "" {
		title = "Edit session"
	}
	marker := func(i int) string {
		if f.focus == i {
			return "› "
		}
		return "  "
	}
	rows := titleStyle.Render(title) + "\n\n"
	for i, in := range f.inputs {
		rows += marker(i) + formLabelStyle.Render(padLabel(inputLabels[i])) + in.View() + "\n"
	}
	cert := len(f.inputs)
	rows += marker(cert) + formLabelStyle.Render(padLabel("cert")) + certLabel(f.ignore) + "\n"
	for i, on := range f.redir {
		rows += marker(cert+1+i) + formLabelStyle.Render(padLabel(redirLabels[i])) + onOff(on) + "\n"
	}
	rows += "\n" + mutedStyle.Render("tab/↑↓ move · space toggle · enter save · esc cancel")
	return boxStyle.Render(rows)
}

func onOff(b bool) string {
	if b {
		return statusOK.Render("on")
	}
	return mutedStyle.Render("off")
}

func padLabel(s string) string {
	for len(s) < 10 {
		s += " "
	}
	return s + " "
}
