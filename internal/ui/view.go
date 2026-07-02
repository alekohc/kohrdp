package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	switch m.state {
	case stateForm:
		return "\n" + m.form.view() + "\n"
	default:
		return m.tableView()
	}
}

func (m Model) tableView() string {
	header := titleStyle.Render("rdpkoh") + "  " + mutedStyle.Render("RDP sessions")
	footer := m.footer()

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		header,
		"",
		m.tableAndLogsView(),
		"",
		footer,
	) + "\n"
}

func (m Model) tableAndLogsView() string {
	tableView := m.table.View()
	if len(m.ordered) == 0 {
		tableView = mutedStyle.Render("\n  No saved sessions. Press n to create one.\n")
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tableView, m.logView())
}

func (m Model) logView() string {
	name, logs, scroll, maxScroll := m.selectedLogs()
	title := "FreeRDP logs"
	if name != "" {
		title += " - " + name
	}

	body := mutedStyle.Render("No logs yet.")
	if len(logs) > 0 {
		rendered := make([]string, len(logs))
		for i, line := range logs {
			rendered[i] = renderLogLine(line)
		}
		body = strings.Join(rendered, "\n")
		if len(logs) < logVisibleLines {
			body += strings.Repeat("\n", logVisibleLines-len(logs))
		}
	} else {
		body += strings.Repeat("\n", logVisibleLines-1)
	}

	meta := mutedStyle.Render("[ and ] scroll")
	if maxScroll > 0 {
		meta = mutedStyle.Render(logScrollLabel(scroll, maxScroll) + "  [ and ] scroll")
	}

	content := titleStyle.Copy().UnsetBackground().Foreground(colorAccent).Render(title)
	content += "\n" + meta + "\n" + body
	style := logBoxStyle
	style = style.Height(logVisibleLines + 2)
	if m.width > 0 {
		tableWidth := lipgloss.Width(m.table.View())
		logWidth := m.width - tableWidth - 6
		if logWidth < 32 {
			logWidth = 32
		}
		style = style.Width(logWidth)
	}

	return style.Render(content)
}

func (m Model) selectedLogs() (string, []string, int, int) {
	n, ok := m.selected()
	if !ok {
		return "", nil, 0, 0
	}
	lf := m.logsBySession[n.Name]
	if lf == nil {
		return n.Name, nil, 0, 0
	}
	scroll := m.logScroll[n.Name]
	maxScroll := m.maxLogScroll(lf.lines)
	if scroll > maxScroll {
		scroll = maxScroll
	}
	logs, err := lf.readWindow(scroll, logVisibleLines)
	if err != nil {
		return n.Name, []string{"[ERROR] failed to read log file: " + err.Error()}, scroll, maxScroll
	}
	return n.Name, logs, scroll, maxScroll
}

func logScrollLabel(scroll, max int) string {
	if max == 0 {
		return "bottom"
	}
	if scroll == 0 {
		return "top"
	}
	if scroll >= max {
		return "bottom"
	}
	return "middle"
}

func renderLogLine(line string) string {
	switch {
	case strings.Contains(line, "[ERROR]"):
		return strings.Replace(line, "[ERROR]", statusErr.Render("[ERROR]"), 1)
	case strings.Contains(line, "[WARN]"):
		return strings.Replace(line, "[WARN]", statusWarn.Render("[WARN]"), 1)
	case strings.Contains(line, "[INFO]"):
		return strings.Replace(line, "[INFO]", statusInfo.Render("[INFO]"), 1)
	case strings.Contains(line, "[DEBUG]"):
		return strings.Replace(line, "[DEBUG]", mutedStyle.Render("[DEBUG]"), 1)
	default:
		return line
	}
}

// footer shows either a modal prompt (delete/cert/password) or the status line
// plus the help bindings.
func (m Model) footer() string {
	switch m.state {
	case stateConfirmDelete:
		return statusWarn.Render("Delete session '"+m.pend.name+"'?  ") +
			mutedStyle.Render("y = yes · any other key = cancel")
	case stateCertConfirm:
		return statusWarn.Render("Ignore certificate errors for "+m.pend.session.Host+"?  ") +
			mutedStyle.Render("y = ignore · n = enforce · esc = cancel")
	case statePassword:
		return formLabelStyle.Render("Password ") + m.pwInput.View() + "\n" +
			mutedStyle.Render("enter = continue · esc = cancel")
	case stateSavePw:
		return statusWarn.Render("Save password to keyring?  ") +
			mutedStyle.Render("y = yes · any other key = no")
	}

	status := ""
	if m.status != "" {
		status = m.stStyle.Render(m.status) + "\n"
	}
	return status + m.help.View(keys)
}
