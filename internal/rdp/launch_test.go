package rdp

import (
	"os"
	"strings"
	"testing"
)

func TestArgsMatchesBashFlags(t *testing.T) {
	os.Setenv("HOME", "/home/test")
	got := strings.Join(Args(Options{User: "admin", Host: "192.168.11.10"}), " ")
	want := `/u:.\admin /from-stdin /v:192.168.11.10 ` +
		`+auto-reconnect +clipboard +fonts /sound /dynamic-resolution ` +
		`/gfx /bpp:32 /drive:Downloads,/home/test/Downloads`
	if got != want {
		t.Errorf("Args mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestArgsOptionalFlags(t *testing.T) {
	args := Args(Options{User: `DOMAIN\u`, Host: "h", WMClass: "work", IgnoreCert: true, Multimon: true})
	joined := strings.Join(args, " ")
	for _, want := range []string{`/u:DOMAIN\u`, "/wm-class:rdpkoh-work", "/multimon", "/cert:ignore"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Args missing %q in: %s", want, joined)
		}
	}
}

func TestArgsConnectionProps(t *testing.T) {
	args := Args(Options{
		User: "u", Host: "h",
		Domain:  "CORP",
		Gateway: "gw.corp.com",
		Size:    "1920x1080",
		Drives:  []string{"work,/home/u/work", "media,/mnt/media"},
	})
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"/d:CORP", "/gateway:g:gw.corp.com", "/size:1920x1080",
		"/drive:work,/home/u/work", "/drive:media,/mnt/media",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("Args missing %q in: %s", want, joined)
		}
	}
	if strings.Contains(joined, "/drive:Downloads,") {
		t.Errorf("explicit drives should replace the default Downloads mount: %s", joined)
	}
}

func TestDisconnectNoMatch(t *testing.T) {
	n, err := Disconnect("nobody", "203.0.113.255")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 processes signalled, got %d", n)
	}
}

func TestLiveKeyMungesUser(t *testing.T) {
	if got, want := LiveKey("admin", "h"), `.\admin`+"\x00"+"h"; got != want {
		t.Errorf("LiveKey = %q, want %q", got, want)
	}
}
