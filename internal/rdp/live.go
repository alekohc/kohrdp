package rdp

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"kohrdp/internal/keyring"
)

type liveProc struct {
	pid int
	key string
}

// scanLive walks /proc and returns every running xfreerdp3 process, tagged with
// the LiveKey of the session it belongs to.
func scanLive() []liveProc {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var procs []liveProc
	for _, e := range entries {
		if !e.IsDir() || e.Name()[0] < '0' || e.Name()[0] > '9' {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(string(data), "\x00")
		if len(args) == 0 || filepath.Base(args[0]) != "xfreerdp3" {
			continue
		}
		var user, host string
		for _, a := range args {
			switch {
			case strings.HasPrefix(a, "/u:"):
				user = strings.TrimPrefix(a, "/u:")
			case strings.HasPrefix(a, "/v:"):
				host = strings.TrimPrefix(a, "/v:")
			}
		}
		if host != "" {
			procs = append(procs, liveProc{pid: pid, key: LiveKey(user, host)})
		}
	}
	return procs
}

// Live returns the set of sessions with a running xfreerdp3 process, keyed by
// LiveKey. It lets a fresh instance discover connections started by a previous
// one.
func Live() map[string]bool {
	live := map[string]bool{}
	for _, p := range scanLive() {
		live[p.key] = true
	}
	return live
}

// Disconnect sends SIGTERM to every running xfreerdp3 process for the session
// (matched by user+host) and returns how many it signalled.
func Disconnect(user, host string) (int, error) {
	want := LiveKey(user, host)
	n := 0
	var firstErr error
	for _, p := range scanLive() {
		if p.key != want {
			continue
		}
		if err := syscall.Kill(p.pid, syscall.SIGTERM); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		n++
	}
	return n, firstErr
}

// LiveKey builds the Live lookup key for a session from its raw user and host,
// normalizing bare usernames to keep domain launches matched to saved sessions.
func LiveKey(user, host string) string {
	return keyring.MungeUser(user) + "\x00" + host
}

// LogDir is the shared directory holding one log file per session, so any
// instance can find the log of a session another instance launched.
func LogDir() string {
	return filepath.Join(os.TempDir(), "kohrdp")
}

// LogPath returns the deterministic log file for a session name.
func LogPath(session string) string {
	safe := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '_', r == '-':
			return r
		default:
			return '_'
		}
	}, session)
	return filepath.Join(LogDir(), safe+".log")
}
