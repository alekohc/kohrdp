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
