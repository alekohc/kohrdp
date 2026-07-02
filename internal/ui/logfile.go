package ui

import (
	"bufio"
	"fmt"
	"os"
)

type sessionLogFile struct {
	path  string
	lines int
}

func (m *Model) ensureLogFile(session string) (*sessionLogFile, error) {
	lf, ok := m.logsBySession[session]
	if ok {
		return lf, nil
	}

	f, err := os.CreateTemp("", "kokoarch-rdp-*.log")
	if err != nil {
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}

	lf = &sessionLogFile{path: f.Name()}
	m.logsBySession[session] = lf
	return lf, nil
}

func (lf *sessionLogFile) append(line string) error {
	f, err := os.OpenFile(lf.path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, line); err != nil {
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
