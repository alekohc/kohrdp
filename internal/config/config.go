// Package config loads and saves the RDP session list from
// ~/.config/kohrdp/sessions.json.
package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Session mirrors one entry in sessions.json. IgnoreCert is a pointer so the
// three states survive a round-trip: nil = never asked, &true = ignore,
// &false = enforce.
type Session struct {
	User       string   `json:"user"`
	Host       string   `json:"host"`
	LastUsed   int64    `json:"lastUsed,omitempty"`
	IgnoreCert *bool    `json:"ignore_cert,omitempty"`
	Drives     []string `json:"drives,omitempty"` // each "name,path"
	Size       string   `json:"size,omitempty"`   // "WxH"
	Domain     string   `json:"domain,omitempty"`
	Gateway    string   `json:"gateway,omitempty"`
	// Redirection toggles. nil = use the default (clipboard/sound on, the rest
	// off), so existing session files keep their behavior.
	Clipboard  *bool `json:"clipboard,omitempty"`
	Sound      *bool `json:"sound,omitempty"`
	Microphone *bool `json:"microphone,omitempty"`
	Printer    *bool `json:"printer,omitempty"`
	Smartcard  *bool `json:"smartcard,omitempty"`
	// LAN trades bandwidth for image quality: lossless codec and no network
	// auto-tuning. nil = off, so existing sessions are unchanged.
	LAN *bool `json:"lan,omitempty"`
}

// BoolOr resolves an optional flag to its effective value.
func BoolOr(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// Named pairs a session with its map key, for ordered display.
type Named struct {
	Name string
	Session
}

// Path returns the config file location.
func Path() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "kohrdp", "sessions.json")
}

// Load reads the session map. A missing file is an empty map, not an error,
// matching the bash launcher seeding '{}'.
func Load() (map[string]Session, error) {
	path := Path()
	if _, err := os.Stat(path); err == nil {
		return LoadFrom(path)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	legacyPath := filepath.Join(os.Getenv("HOME"), ".config", "rdpkoh", "sessions.json")
	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return map[string]Session{}, nil
	} else if err != nil {
		return nil, err
	}

	sessions, err := LoadFrom(legacyPath)
	if err != nil {
		return nil, err
	}
	if err := SaveTo(path, sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

func LoadFrom(path string) (map[string]Session, error) {
	if path == "-" {
		return LoadReader(os.Stdin)
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	return LoadReaderBytes(data)
}

func LoadReader(r io.Reader) (map[string]Session, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return LoadReaderBytes(data)
}

func LoadReaderBytes(data []byte) (map[string]Session, error) {
	sessions := map[string]Session{}
	if len(data) == 0 {
		return sessions, nil
	}
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// Save writes the session map atomically (write temp, rename).
func Save(sessions map[string]Session) error {
	return SaveTo(Path(), sessions)
}

func SaveTo(path string, sessions map[string]Session) error {
	if path == "-" {
		return SaveWriter(os.Stdout, sessions)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := marshal(sessions)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func SaveWriter(w io.Writer, sessions map[string]Session) error {
	data, err := marshal(sessions)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func marshal(sessions map[string]Session) ([]byte, error) {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return data, nil
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
