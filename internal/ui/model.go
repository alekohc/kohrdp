package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"kokoarch-rdp/internal/config"
	"kokoarch-rdp/internal/keyring"
	"kokoarch-rdp/internal/rdp"
)

type state int

const maxLogs = 8

const (
	stateTable state = iota
	stateForm
	stateConfirmDelete
	stateCertConfirm
	statePassword
	stateSavePw
)

// pending carries a connection through the cert → password → launch steps.
type pending struct {
	name     string
	session  config.Session
	multimon bool
	password string
}

// Model is the root bubbletea model.
type Model struct {
	sessions      map[string]config.Session
	ordered       []config.Named // parallel to table rows
	table         table.Model
	help          help.Model
	form          form
	pwInput       textinput.Model
	state         state
	pend          pending
	status        string
	stStyle       lipgloss.Style
	activeSession map[string]bool
	logsBySession map[string][]string
	rdpMsgCh      chan tea.Msg
	width         int
}

type rdpLogMsg struct {
	session string
	line    string
}

type rdpDoneMsg struct {
	session string
}

// New builds the initial model from the loaded session map.
func New(sessions map[string]config.Session) Model {
	t := table.New(
		table.WithColumns(columns()),
		table.WithFocused(true),
		table.WithHeight(12),
	)
	t.SetStyles(tableStyles())

	pw := textinput.New()
	pw.EchoMode = textinput.EchoPassword
	pw.Placeholder = "password"
	pw.CharLimit = 256

	m := Model{
		sessions:      sessions,
		table:         t,
		help:          help.New(),
		pwInput:       pw,
		state:         stateTable,
		stStyle:       mutedStyle,
		activeSession: make(map[string]bool),
		logsBySession: make(map[string][]string),
		rdpMsgCh:      make(chan tea.Msg, 64),
	}
	m.reload()
	return m
}

func columns() []table.Column {
	return []table.Column{
		{Title: "Name", Width: 18},
		{Title: "User@Host", Width: 30},
		{Title: "Status", Width: 8},
		{Title: "Cert", Width: 8},
		{Title: "Last used", Width: 10},
	}
}

func tableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).Foreground(colorAccent).
		BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorMuted).BorderBottom(true)
	s.Selected = s.Selected.Bold(true).Foreground(colorText).Background(colorAccent)
	return s
}

// reload rebuilds the ordered slice and table rows from the session map.
func (m *Model) reload() {
	m.ordered = config.SortedByRecency(m.sessions)
	rows := make([]table.Row, len(m.ordered))
	for i, n := range m.ordered {
		rows[i] = table.Row{
			n.Name,
			n.User + "@" + n.Host,
			sessionStatus(m.activeSession[n.Name]),
			certLabel(n.IgnoreCert),
			relTime(n.LastUsed),
		}
	}
	m.table.SetRows(rows)
}

func (m Model) selected() (config.Named, bool) {
	if len(m.ordered) == 0 {
		return config.Named{}, false
	}
	return m.ordered[m.table.Cursor()], true
}

func (m *Model) setStatus(s string, style lipgloss.Style) {
	m.status = s
	m.stStyle = style
}

func (m Model) Init() tea.Cmd { return waitRDPMsg(m.rdpMsgCh) }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.help.Width = msg.Width
		return m, nil
	case tea.KeyMsg:
		switch m.state {
		case stateForm:
			return m.updateForm(msg)
		case stateConfirmDelete:
			return m.updateConfirmDelete(msg)
		case stateCertConfirm:
			return m.updateCertConfirm(msg)
		case statePassword:
			return m.updatePassword(msg)
		case stateSavePw:
			return m.updateSavePw(msg)
		default:
			return m.updateTable(msg)
		}
	case rdpLogMsg:
		m.appendLog(msg.session, msg.line)
		return m, waitRDPMsg(m.rdpMsgCh)
	case rdpDoneMsg:
		delete(m.activeSession, msg.session)
		m.reload()
		return m, waitRDPMsg(m.rdpMsgCh)
	}
	return m, nil
}

func waitRDPMsg(ch <-chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) appendLog(session, line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	logs := append(m.logsBySession[session], line)
	if len(logs) > maxLogs {
		logs = logs[len(logs)-maxLogs:]
	}
	m.logsBySession[session] = logs
}

// --- table view ---

func (m Model) updateTable(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, keys.New):
		m.form = newForm()
		m.state = stateForm
		return m, textinput.Blink
	case key.Matches(msg, keys.Edit):
		if n, ok := m.selected(); ok {
			m.form = editForm(n.Name, n.User, n.Host, n.IgnoreCert)
			m.state = stateForm
			return m, textinput.Blink
		}
	case key.Matches(msg, keys.Delete):
		if n, ok := m.selected(); ok {
			m.pend = pending{name: n.Name}
			m.state = stateConfirmDelete
		}
		return m, nil
	case key.Matches(msg, keys.Cert):
		if n, ok := m.selected(); ok {
			s := n.Session
			s.IgnoreCert = cycleCert(s.IgnoreCert)
			m.sessions[n.Name] = s
			if err := config.Save(m.sessions); err != nil {
				m.setStatus("save failed: "+err.Error(), statusErr)
			} else {
				m.setStatus(fmt.Sprintf("%s cert: %s", n.Name, certLabel(s.IgnoreCert)), statusOK)
			}
			m.reload()
		}
		return m, nil
	case key.Matches(msg, keys.ClearPw):
		if n, ok := m.selected(); ok {
			if err := keyring.Clear(n.Host, n.User); err != nil {
				m.setStatus("clear failed: "+err.Error(), statusErr)
			} else {
				m.setStatus("password cleared for "+n.Name, statusOK)
			}
		}
		return m, nil
	case key.Matches(msg, keys.Multimon):
		if n, ok := m.selected(); ok {
			return m.startConnect(n, true)
		}
		return m, nil
	case key.Matches(msg, keys.Connect):
		if n, ok := m.selected(); ok {
			return m.startConnect(n, false)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// --- form view ---

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmd, done, submitted := m.form.update(msg)
	if !done {
		return m, cmd
	}
	if submitted {
		name := m.form.name()
		if m.form.origName != "" && m.form.origName != name {
			delete(m.sessions, m.form.origName)
			if logs, ok := m.logsBySession[m.form.origName]; ok {
				m.logsBySession[name] = logs
				delete(m.logsBySession, m.form.origName)
			}
			if m.activeSession[m.form.origName] {
				m.activeSession[name] = true
				delete(m.activeSession, m.form.origName)
			}
		}
		s := m.sessions[name]
		s.User = m.form.user()
		s.Host = m.form.host()
		s.IgnoreCert = m.form.ignore
		m.sessions[name] = s
		if err := config.Save(m.sessions); err != nil {
			m.setStatus("save failed: "+err.Error(), statusErr)
		} else {
			m.setStatus("saved "+name, statusOK)
		}
		m.reload()
	}
	m.state = stateTable
	return m, nil
}

// --- delete confirm ---

func (m Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		delete(m.sessions, m.pend.name)
		delete(m.activeSession, m.pend.name)
		delete(m.logsBySession, m.pend.name)
		if err := config.Save(m.sessions); err != nil {
			m.setStatus("save failed: "+err.Error(), statusErr)
		} else {
			m.setStatus("deleted "+m.pend.name, statusOK)
		}
		m.reload()
	}
	m.state = stateTable
	return m, nil
}

// --- connect flow ---

func (m Model) startConnect(n config.Named, multimon bool) (tea.Model, tea.Cmd) {
	m.pend = pending{name: n.Name, session: n.Session, multimon: multimon}
	if n.IgnoreCert == nil {
		m.state = stateCertConfirm
		return m, nil
	}
	return m.resolvePassword()
}

func (m Model) updateCertConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		t := true
		m.pend.session.IgnoreCert = &t
	case "esc":
		m.state = stateTable
		return m, nil
	default:
		f := false
		m.pend.session.IgnoreCert = &f
	}
	m.sessions[m.pend.name] = m.pend.session
	_ = config.Save(m.sessions)
	m.reload()
	return m.resolvePassword()
}

// resolvePassword looks up the keyring; on a hit it connects, on a miss it opens
// the in-TUI password prompt.
func (m Model) resolvePassword() (tea.Model, tea.Cmd) {
	pw, err := keyring.Lookup(m.pend.session.Host, m.pend.session.User)
	if err != nil {
		m.setStatus("keyring error: "+err.Error(), statusErr)
		m.state = stateTable
		return m, nil
	}
	if pw != "" {
		m.pend.password = pw
		return m.finishConnect()
	}
	m.pwInput.SetValue("")
	m.pwInput.Focus()
	m.state = statePassword
	return m, textinput.Blink
}

func (m Model) updatePassword(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.pwInput.SetValue("")
		m.state = stateTable
		return m, nil
	case "enter":
		m.pend.password = m.pwInput.Value()
		m.pwInput.SetValue("")
		m.state = stateSavePw
		return m, nil
	}
	var cmd tea.Cmd
	m.pwInput, cmd = m.pwInput.Update(msg)
	return m, cmd
}

func (m Model) updateSavePw(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if err := keyring.Store(m.pend.session.Host, m.pend.session.User, m.pend.password); err != nil {
			m.setStatus("keyring store failed: "+err.Error(), statusErr)
		}
	}
	return m.finishConnect()
}

// finishConnect bumps recency, persists, spawns xfreerdp3 detached, then wipes
// the in-memory password.
func (m Model) finishConnect() (tea.Model, tea.Cmd) {
	m.pend.session.LastUsed = time.Now().Unix()
	m.sessions[m.pend.name] = m.pend.session
	_ = config.Save(m.sessions)

	ignore := m.pend.session.IgnoreCert != nil && *m.pend.session.IgnoreCert
	sessionName := m.pend.name
	err := rdp.Launch(rdp.Options{
		User:       m.pend.session.User,
		Host:       m.pend.session.Host,
		IgnoreCert: ignore,
		Multimon:   m.pend.multimon,
	}, m.pend.password, func(line string) {
		m.rdpMsgCh <- rdpLogMsg{session: sessionName, line: line}
	}, func() {
		m.rdpMsgCh <- rdpDoneMsg{session: sessionName}
	})

	m.pend.password = "" // wipe secret as soon as it is handed off
	if err != nil {
		m.setStatus("launch failed: "+err.Error(), statusErr)
	} else {
		m.activeSession[m.pend.name] = true
		m.setStatus("connecting to "+m.pend.name, statusOK)
	}
	m.reload()
	m.state = stateTable
	return m, nil
}
