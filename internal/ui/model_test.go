package ui

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"rdpkoh/internal/config"
)

func cleanupModelLogs(t *testing.T, m *Model) {
	t.Helper()
	t.Cleanup(func() {
		for _, lf := range m.logsBySession {
			if lf != nil {
				_ = os.Remove(lf.path)
			}
		}
	})
}

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

// Starting empty leaves the table cursor at -1; adding the first session must
// not leave it there (would panic in selected) and should select row 0.
func TestFirstSessionAfterEmptySelectsRow(t *testing.T) {
	m := New(map[string]config.Session{})
	m.sessions["tp-test"] = config.Session{User: "administrator", Host: "192.168.11.10"}
	m.reload()

	if _, ok := m.selected(); !ok {
		t.Fatal("no session selected after adding first session")
	}
	if out := m.View(); !strings.Contains(out, "tp-test") {
		t.Errorf("view missing added session:\n%s", out)
	}
}

func TestLogViewRendersRecentLines(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
		"b": {User: "u", Host: "h2", LastUsed: 1},
	})
	cleanupModelLogs(t, &m)
	for i := 0; i < logVisibleLines+2; i++ {
		m.appendLog("a", "line")
	}

	if got := m.logsBySession["a"].lines; got != logVisibleLines+2 {
		t.Fatalf("expected %d logs, got %d", logVisibleLines+2, got)
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
	cleanupModelLogs(t, &m)
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

func TestSelectedLogsRespectsScroll(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
	})
	cleanupModelLogs(t, &m)
	for i := 0; i < logVisibleLines+3; i++ {
		m.appendLog("a", strings.Repeat("x", i+1))
	}

	_, logs, scroll, maxScroll := m.selectedLogs()
	if scroll != maxScroll {
		t.Fatalf("expected autoscroll to bottom, got scroll=%d max=%d", scroll, maxScroll)
	}
	if len(logs) != logVisibleLines {
		t.Fatalf("expected %d visible log lines, got %d", logVisibleLines, len(logs))
	}

	m.scrollLogs("a", -logVisibleLines/2)
	_, logs, scroll, _ = m.selectedLogs()
	if scroll >= maxScroll {
		t.Fatalf("expected scroll to move upward, got %d", scroll)
	}
	if len(logs) != logVisibleLines {
		t.Fatalf("expected %d visible log lines after scroll, got %d", logVisibleLines, len(logs))
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

func TestRDPDoneMarksSessionExited(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
	})
	m.sessionState["a"] = sessionActive
	m.reload()

	updated, _ := m.Update(rdpDoneMsg{session: "a"})
	m2 := updated.(Model)

	if got := m2.sessionState["a"]; got != sessionExited {
		t.Fatalf("expected sessionExited, got %v", got)
	}
	if out := m2.View(); !strings.Contains(out, "exited") {
		t.Errorf("expected exited status in view:\n%s", out)
	}
}

func TestRDPDoneMarksSessionFailed(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
	})
	m.sessionState["a"] = sessionActive
	m.reload()

	updated, _ := m.Update(rdpDoneMsg{session: "a", err: errors.New("boom")})
	m2 := updated.(Model)

	if got := m2.sessionState["a"]; got != sessionFailed {
		t.Fatalf("expected sessionFailed, got %v", got)
	}
	if out := m2.View(); !strings.Contains(out, "failed") {
		t.Errorf("expected failed status in view:\n%s", out)
	}
}

func TestRDPActiveMarksSessionActive(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
	})
	m.sessionState["a"] = sessionStarting
	m.reload()

	updated, _ := m.Update(rdpActiveMsg{session: "a"})
	m2 := updated.(Model)

	if got := m2.sessionState["a"]; got != sessionActive {
		t.Fatalf("expected sessionActive, got %v", got)
	}
	if out := m2.View(); !strings.Contains(out, "active") {
		t.Errorf("expected active status in view:\n%s", out)
	}
}

func TestCleanupRemovesLogFiles(t *testing.T) {
	m := New(map[string]config.Session{
		"a": {User: "u", Host: "h1", LastUsed: 2},
	})
	m.appendLog("a", "alpha")
	lf := m.logsBySession["a"]
	if lf == nil {
		t.Fatal("expected log file to exist")
	}
	if _, err := os.Stat(lf.path); err != nil {
		t.Fatalf("expected log file on disk: %v", err)
	}

	m.Cleanup()
	if _, err := os.Stat(lf.path); !os.IsNotExist(err) {
		t.Fatalf("expected log file removed, got err=%v", err)
	}
}
