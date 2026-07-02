// Package config loads and saves the RDP session list, staying byte-compatible
// with the bash launcher's ~/.config/kokoarch/rdp-sessions.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Session mirrors one entry in rdp-sessions.json. IgnoreCert is a pointer so the
// three states the bash script relies on survive a round-trip: nil = never
// asked, &true = ignore, &false = enforce.
type Session struct {
	User       string `json:"user"`
	Host       string `json:"host"`
	LastUsed   int64  `json:"lastUsed,omitempty"`
	IgnoreCert *bool  `json:"ignore_cert,omitempty"`
}

// Named pairs a session with its map key, for ordered display.
type Named struct {
	Name string
	Session
}

// Path returns the config file location, honouring the same path the bash
// launcher uses.
func Path() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "kokoarch", "rdp-sessions.json")
}

// Load reads the session map. A missing file is an empty map, not an error,
// matching the bash launcher seeding '{}'.
func Load() (map[string]Session, error) {
	data, err := os.ReadFile(Path())
	if os.IsNotExist(err) {
		return map[string]Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	sessions := map[string]Session{}
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// Save writes the session map atomically (write temp, rename), the same way the
// bash launcher does via "$RDP_CONFIG.tmp" && mv.
func Save(sessions map[string]Session) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SortedByRecency returns sessions most-recently-used first, matching the bash
// picker's "sort_by(.lastUsed) | reverse".
func SortedByRecency(sessions map[string]Session) []Named {
	out := make([]Named, 0, len(sessions))
	for name, s := range sessions {
		out = append(out, Named{Name: name, Session: s})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].LastUsed > out[j].LastUsed
	})
	return out
}
