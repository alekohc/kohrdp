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
	for _, want := range []string{`/u:DOMAIN\u`, "/wm-class:kohrdp-work", "/multimon", "/cert:ignore"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Args missing %q in: %s", want, joined)
		}
	}
}

func TestArgsUsernameDomain(t *testing.T) {
	for _, tt := range []struct {
		name, user, domain, want string
	}{
		{"local", "admin", "", `/u:.\admin`},
		{"domain", "administratör", "norrlands.com", "/u:administratör"},
		{"qualified", `NORRLANDS\administratör`, "", `/u:NORRLANDS\administratör`},
		{"qualified with domain", `NORRLANDS\administratör`, "norrlands.com", `/u:NORRLANDS\administratör`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args := Args(Options{User: tt.user, Host: "h", Domain: tt.domain})
			if args[0] != tt.want {
				t.Errorf("username argument = %q, want %q", args[0], tt.want)
			}
			var domain string
			for _, arg := range args {
				if strings.HasPrefix(arg, "/d:") {
					domain = strings.TrimPrefix(arg, "/d:")
				}
			}
			if domain != tt.domain {
				t.Errorf("domain = %q, want %q", domain, tt.domain)
			}
			if got := LiveKey(strings.TrimPrefix(args[0], "/u:"), "h"); got != LiveKey(tt.user, "h") {
				t.Errorf("launched username does not match saved session: %q", got)
			}
		})
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

func TestArgsRedirection(t *testing.T) {
	def := strings.Join(Args(Options{User: "u", Host: "h"}), " ")
	if !strings.Contains(def, "+clipboard") || !strings.Contains(def, "/sound") {
		t.Errorf("defaults should keep clipboard and sound: %s", def)
	}
	for _, off := range []string{"/microphone", "/printer", "/smartcard", "-clipboard"} {
		if strings.Contains(def, off) {
			t.Errorf("defaults should not contain %q: %s", off, def)
		}
	}

	on := strings.Join(Args(Options{
		User: "u", Host: "h",
		NoClipboard: true, NoSound: true,
		Microphone: true, Printer: true, Smartcard: true,
	}), " ")
	for _, want := range []string{"-clipboard", "/microphone", "/printer", "/smartcard"} {
		if !strings.Contains(on, want) {
			t.Errorf("toggled args missing %q: %s", want, on)
		}
	}
	if strings.Contains(on, "/sound") || strings.Contains(on, "+clipboard") {
		t.Errorf("sound/clipboard should be disabled: %s", on)
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

func TestArgsLAN(t *testing.T) {
	lan := strings.Join(Args(Options{User: "u", Host: "h", LAN: true}), " ")
	for _, want := range []string{"/network:lan", "/gfx:AVC420:off,AVC444:off"} {
		if !strings.Contains(lan, want) {
			t.Errorf("missing %q in %s", want, lan)
		}
	}
	if strings.Contains(lan, "/gfx ") {
		t.Errorf("plain /gfx should be replaced: %s", lan)
	}
	off := strings.Join(Args(Options{User: "u", Host: "h"}), " ")
	if strings.Contains(off, "/network") || strings.Contains(off, "AVC") {
		t.Errorf("LAN off must not change flags: %s", off)
	}
}

func TestArgsScale(t *testing.T) {
	for scale, want := range map[string]string{
		"140": "/scale:140",
		"125": "/scale-desktop:125",
	} {
		got := strings.Join(Args(Options{User: "u", Host: "h", Scale: scale}), " ")
		if !strings.Contains(got, want) {
			t.Errorf("scale %s: missing %q in %s", scale, want, got)
		}
	}
	if got := strings.Join(Args(Options{User: "u", Host: "h"}), " "); strings.Contains(got, "scale") {
		t.Errorf("empty scale must emit nothing: %s", got)
	}
}
