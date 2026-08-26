package ui

import (
	"bufio"
	"os"

	"kohrdp/internal/rdp"
)

type sessionLogFile struct {
	path  string
	lines int
}

func (m *Model) ensureLogFile(session string) (*sessionLogFile, error) {
	if lf, ok := m.logsBySession[session]; ok {
		return lf, nil
	}
	if err := os.MkdirAll(rdp.LogDir(), 0o700); err != nil {
		return nil, err
	}

	path := rdp.LogPath(session)
	lf := &sessionLogFile{path: path, lines: countLines(path)}
	m.logsBySession[session] = lf
	return lf, nil
}

func (lf *sessionLogFile) append(line string) error {
	f, err := os.OpenFile(lf.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(line + "\n"); err != nil {
		return err
	}
	lf.lines++
	return nil
}

func (lf *sessionLogFile) readWindow(start, size int) ([]string, error) {
	if size <= 0 {
		return nil, nil
	}

	f, err := os.Open(lf.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines := make([]string, 0, size)
	s := bufio.NewScanner(f)
	idx := 0
	for s.Scan() {
		if idx >= start {
			lines = append(lines, s.Text())
			if len(lines) == size {
				break
			}
		}
		idx++
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func (lf *sessionLogFile) remove() {
	if lf == nil || lf.path == "" {
		return
	}
	_ = os.Remove(lf.path)
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	n := 0
	s := bufio.NewScanner(f)
	for s.Scan() {
		n++
	}
	return n
}

// Cleanup removes log files on quit, but keeps those of sessions that are still
// running so the next instance can pick their logs back up.
func (m Model) Cleanup() {
	live := rdp.Live()
	for name, lf := range m.logsBySession {
		if lf == nil {
			continue
		}
		s := m.sessions[name]
		if live[rdp.LiveKey(s.User, s.Host)] {
			continue
		}
		lf.remove()
	}
}
