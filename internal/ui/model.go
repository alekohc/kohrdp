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

	"rdpkoh/internal/config"
	"rdpkoh/internal/keyring"
	"rdpkoh/internal/rdp"
)

type state int

const (
	logVisibleLines = 12
)

type sessionState int

const (
	sessionIdle sessionState = iota
	sessionStarting
	sessionActive
	sessionExited
	sessionFailed
)

const (
	stateTable state = iota
	stateForm
	stateConfirmDelete
	stateConfirmDisconnect
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
	sessionState  map[string]sessionState
	logsBySession map[string]*sessionLogFile
	logScroll     map[string]int
	rdpMsgCh      chan tea.Msg
	width         int
}

type rdpDoneMsg struct {
	session string
	err     error
}

type tickMsg struct{}

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
		sessionState:  make(map[string]sessionState),
		logsBySession: make(map[string]*sessionLogFile),
		logScroll:     make(map[string]int),
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
			sessionStatus(m.sessionState[n.Name]),
			certLabel(n.IgnoreCert),
			relTime(n.LastUsed),
		}
	}
	m.table.SetRows(rows)
	if m.table.Cursor() < 0 && len(rows) > 0 {
		m.table.SetCursor(0)
	}
}

func (m Model) selected() (config.Named, bool) {
	i := m.table.Cursor()
	if i < 0 || i >= len(m.ordered) {
		return config.Named{}, false
	}
	return m.ordered[i], true
}

func (m *Model) setStatus(s string, style lipgloss.Style) {
	m.status = s
	m.stStyle = style
}

func (m Model) Init() tea.Cmd {
	// Scan for already-running sessions right away, then on every tick.
	return tea.Batch(waitRDPMsg(m.rdpMsgCh), func() tea.Msg { return tickMsg{} })
}

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
		case stateConfirmDisconnect:
			return m.updateConfirmDisconnect(msg)
		case stateCertConfirm:
			return m.updateCertConfirm(msg)
		case statePassword:
			return m.updatePassword(msg)
		case stateSavePw:
			return m.updateSavePw(msg)
		default:
			return m.updateTable(msg)
		}
	case rdpDoneMsg:
		if msg.err != nil {
			m.sessionState[msg.session] = sessionFailed
		} else {
			m.sessionState[msg.session] = sessionExited
		}
		m.reload()
		return m, waitRDPMsg(m.rdpMsgCh)
	case tickMsg:
		m.refreshLiveStates()
		for session := range m.logsBySession {
			m.syncLogCount(session)
		}
		m.reload()
		return m, tick()
	}
	return m, nil
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// refreshLiveStates reconciles session status with the xfreerdp3 processes
// actually running, so sessions started by a previous instance show as active.
func (m *Model) refreshLiveStates() {
	m.applyLive(rdp.Live())
}

func (m *Model) applyLive(live map[string]bool) {
	for name, s := range m.sessions {
		if live[rdp.LiveKey(s.User, s.Host)] {
			m.sessionState[name] = sessionActive
			m.ensureLogFile(name)
		} else if st := m.sessionState[name]; st == sessionActive || st == sessionStarting {
			m.sessionState[name] = sessionExited
		}
	}
}

// syncLogCount recounts a session's log file (the child writes it directly) and
// keeps the view pinned to the bottom if it already was.
func (m *Model) syncLogCount(session string) {
	lf := m.logsBySession[session]
	if lf == nil {
		return
	}
	total := countLines(lf.path)
	if total == lf.lines {
		return
	}
	wasAtBottom := m.logScroll[session] >= m.maxLogScroll(lf.lines)
	lf.lines = total
	if wasAtBottom {
		m.logScroll[session] = m.maxLogScroll(total)
	}
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
	lf, err := m.ensureLogFile(session)
	if err != nil {
		m.setStatus("log file failed: "+err.Error(), statusErr)
		return
	}
	wasAtBottom := m.logScroll[session] >= m.maxLogScroll(lf.lines)
	if err := lf.append(line); err != nil {
		m.setStatus("log write failed: "+err.Error(), statusErr)
		return
	}
	if wasAtBottom {
		m.logScroll[session] = m.maxLogScroll(lf.lines)
	}
}

func (m Model) maxLogScroll(total int) int {
	if total <= logVisibleLines {
		return 0
	}
	return total - logVisibleLines
}

func (m *Model) scrollLogs(session string, delta int) {
	max := 0
	if lf := m.logsBySession[session]; lf != nil {
		max = m.maxLogScroll(lf.lines)
	}
	next := m.logScroll[session] + delta
	if next < 0 {
		next = 0
	}
	if next > max {
		next = max
	}
	m.logScroll[session] = next
}

// --- table view ---

func (m Model) updateTable(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, keys.LogOlder):
		if n, ok := m.selected(); ok {
			m.scrollLogs(n.Name, -logVisibleLines/2)
		}
		return m, nil
	case key.Matches(msg, keys.LogNewer):
		if n, ok := m.selected(); ok {
			m.scrollLogs(n.Name, logVisibleLines/2)
		}
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
	case key.Matches(msg, keys.Disconnect):
		if n, ok := m.selected(); ok {
			st := m.sessionState[n.Name]
			if st != sessionActive && st != sessionStarting {
				m.setStatus(n.Name+" is not connected", statusWarn)
				return m, nil
			}
			m.pend = pending{name: n.Name, session: n.Session}
			m.state = stateConfirmDisconnect
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
			if lf, ok := m.logsBySession[m.form.origName]; ok {
				m.logsBySession[name] = lf
				delete(m.logsBySession, m.form.origName)
			}
			if state, ok := m.sessionState[m.form.origName]; ok {
				m.sessionState[name] = state
				delete(m.sessionState, m.form.origName)
			}
			if scroll, ok := m.logScroll[m.form.origName]; ok {
				m.logScroll[name] = scroll
				delete(m.logScroll, m.form.origName)
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
		delete(m.sessionState, m.pend.name)
		if lf := m.logsBySession[m.pend.name]; lf != nil {
			lf.remove()
		}
		delete(m.logsBySession, m.pend.name)
		delete(m.logScroll, m.pend.name)
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

// --- disconnect confirm ---

func (m Model) updateConfirmDisconnect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		count, err := rdp.Disconnect(m.pend.session.User, m.pend.session.Host)
		switch {
		case err != nil:
			m.setStatus("disconnect failed: "+err.Error(), statusErr)
		case count == 0:
			m.setStatus(m.pend.name+" is not connected", statusWarn)
		default:
			m.setStatus("disconnecting "+m.pend.name, statusOK)
		}
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
	lf, err := m.ensureLogFile(sessionName)
	if err != nil {
		m.pend.password = ""
		m.sessionState[sessionName] = sessionFailed
		m.setStatus("log file failed: "+err.Error(), statusErr)
		m.reload()
		m.state = stateTable
		return m, nil
	}

	err = rdp.Launch(rdp.Options{
		User:       m.pend.session.User,
		Host:       m.pend.session.Host,
		IgnoreCert: ignore,
		Multimon:   m.pend.multimon,
	}, m.pend.password, lf.path, func(err error) {
		m.rdpMsgCh <- rdpDoneMsg{session: sessionName, err: err}
	})

	m.pend.password = "" // wipe secret as soon as it is handed off
	if err != nil {
		m.sessionState[m.pend.name] = sessionFailed
		m.setStatus("launch failed: "+err.Error(), statusErr)
	} else {
		m.sessionState[m.pend.name] = sessionStarting
		lf.lines = 0 // Launch truncated the log for the fresh connection
		m.logScroll[m.pend.name] = 0
		m.setStatus("connecting to "+m.pend.name, statusOK)
	}
	m.reload()
	m.state = stateTable
	return m, nil
}
