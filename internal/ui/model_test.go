package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"kokoarch-rdp/internal/config"
)

func TestViewRenders(t *testing.T) {
	yes := true
	m := New(map[string]config.Session{
		"tp-test": {User: "administrator", Host: "192.168.11.10", LastUsed: 1782478828, IgnoreCert: &yes},
	})

	if out := m.View(); !strings.Contains(out, "tp-test") || !strings.Contains(out, "administrator@192.168.11.10") {
		t.Errorf("table view missing session row:\n%s", out)
	}
	if out := m.View(); !strings.Contains(out, "idle") {
		t.Errorf("table view missing idle status:\n%s", out)
	}

	m.form = newForm()
	m.state = stateForm
	if out := m.View(); !strings.Contains(out, "New session") {
		t.Errorf("form view not rendered:\n%s", out)
	}
}

func TestEmptyViewRenders(t *testing.T) {
	m := New(map[string]config.Session{})
	if out := m.View(); !strings.Contains(out, "No saved sessions") {
		t.Errorf("empty view missing hint:\n%s", out)
	}
}

func TestLogViewRendersRecentLines(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
		"b": {User: "u", Host: "h2", LastUsed: 1},
	})
	for i := 0; i < maxLogs+2; i++ {
		m.appendLog("a", "line")
	}

	if len(m.logsBySession["a"]) != maxLogs {
		t.Fatalf("expected %d logs, got %d", maxLogs, len(m.logsBySession["a"]))
	}
	if out := m.View(); !strings.Contains(out, "line") {
		t.Errorf("log view missing lines:\n%s", out)
	}
}

func TestLogViewTracksSelectedSession(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
		"b": {User: "u", Host: "h2", LastUsed: 1},
	})
	m.appendLog("a", "alpha")
	m.appendLog("b", "beta")

	if out := m.View(); !strings.Contains(out, "alpha") || strings.Contains(out, "beta") {
		t.Errorf("selected log view mismatch for first session:\n%s", out)
	}

	m.table.SetCursor(1)
	if out := m.View(); !strings.Contains(out, "beta") || strings.Contains(out, "alpha") {
		t.Errorf("selected log view mismatch after moving cursor:\n%s", out)
	}
}

func TestRenderLogLineHighlightsSeverity(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(termenv.TrueColor)
	})

	out := renderLogLine("[ERROR] boom")
	if !strings.Contains(out, "[ERROR]") || !strings.Contains(out, "\x1b[") {
		t.Errorf("expected colored error log, got %q", out)
	}

	out = renderLogLine("[WARN] heads up")
	if !strings.Contains(out, "[WARN]") || !strings.Contains(out, "\x1b[") {
		t.Errorf("expected colored warn log, got %q", out)
	}
}

func TestRDPDoneMarksSessionIdle(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
	})
	m.activeSession["a"] = true
	m.reload()

	updated, _ := m.Update(rdpDoneMsg{session: "a"})
	m2 := updated.(Model)

	if m2.activeSession["a"] {
		t.Fatal("expected session to be idle after rdpDoneMsg")
	}
	if out := m2.View(); !strings.Contains(out, "idle") {
		t.Errorf("expected idle status in view:\n%s", out)
	}
}
