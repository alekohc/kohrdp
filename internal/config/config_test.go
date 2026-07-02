package config

import (
	"path/filepath"
	"testing"
)

func TestSaveToAndLoadFromRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	ignore := true
	in := map[string]Session{
		"work": {User: "admin", Host: "host", LastUsed: 123, IgnoreCert: &ignore},
	}

	if err := SaveTo(path, in); err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}

	out, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("LoadFrom() len = %d, want 1", len(out))
	}
	if got := out["work"]; got.User != in["work"].User || got.Host != in["work"].Host || got.LastUsed != in["work"].LastUsed || got.IgnoreCert == nil || *got.IgnoreCert != true {
		t.Fatalf("LoadFrom() = %#v, want %#v", got, in["work"])
	}
}
