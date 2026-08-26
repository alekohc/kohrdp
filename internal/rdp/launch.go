// Package rdp builds the xfreerdp3 invocation and spawns it, porting the bash
// launcher's flag set and stdin password handling verbatim.
package rdp

import (
	"os"
	"os/exec"
	"strings"

	"kohrdp/internal/config"
	"kohrdp/internal/keyring"
)

// Options mirrors the bash launcher's tunable flags.
type Options struct {
	User       string // raw username; munged here for /u:
	Host       string
	WMClass    string // bare name, e.g. "work"; "" to omit
	IgnoreCert bool
	Multimon   bool
	Domain     string   // /d: ; "" to omit
	Gateway    string   // /gateway:g: ; "" to omit
	Size       string   // /size:WxH ; "" to omit
	Drives     []string // each "name,path"; empty => default Downloads mount
	// Redirection. Clipboard and sound are on unless disabled; the rest are off
	// unless enabled — so the zero value keeps the historical behavior.
	NoClipboard bool
	NoSound     bool
	Microphone  bool
	Printer     bool
	Smartcard   bool
}

// OptionsFromSession maps a stored session (plus the raw user/host and a
// per-launch multimon choice) to launch Options, applying the redirection
// defaults. Shared by the TUI and the CLI so both build the same command.
func OptionsFromSession(user, host string, s config.Session, multimon bool) Options {
	return Options{
		User:        user,
		Host:        host,
		IgnoreCert:  config.BoolOr(s.IgnoreCert, false),
		Multimon:    multimon,
		Domain:      s.Domain,
		Gateway:     s.Gateway,
		Size:        s.Size,
		Drives:      s.Drives,
		NoClipboard: !config.BoolOr(s.Clipboard, true),
		NoSound:     !config.BoolOr(s.Sound, true),
		Microphone:  config.BoolOr(s.Microphone, false),
		Printer:     config.BoolOr(s.Printer, false),
		Smartcard:   config.BoolOr(s.Smartcard, false),
	}
}

// Args returns the xfreerdp3 argument vector, in the same order as the bash
// launcher. Exposed for testing so the flag set can be asserted without a spawn.
func Args(o Options) []string {
	args := []string{
		"/u:" + keyring.MungeUser(o.User),
		"/from-stdin",
		"/v:" + o.Host,
	}
	if o.Domain != "" {
		args = append(args, "/d:"+o.Domain)
	}
	if o.WMClass != "" {
		args = append(args, "/wm-class:kohrdp-"+o.WMClass)
	}
	if o.Multimon {
		args = append(args, "/multimon")
	}
	if o.IgnoreCert {
		args = append(args, "/cert:ignore")
	}
	if o.Gateway != "" {
		args = append(args, "/gateway:g:"+o.Gateway)
	}
	if o.Size != "" {
		args = append(args, "/size:"+o.Size)
	}
	args = append(args, "+auto-reconnect")
	if o.NoClipboard {
		args = append(args, "-clipboard")
	} else {
		args = append(args, "+clipboard")
	}
	args = append(args, "+fonts")
	if !o.NoSound {
		args = append(args, "/sound")
	}
	args = append(args, "/dynamic-resolution", "/gfx", "/bpp:32")
	if o.Microphone {
		args = append(args, "/microphone")
	}
	if o.Printer {
		args = append(args, "/printer")
	}
	if o.Smartcard {
		args = append(args, "/smartcard")
	}
	if len(o.Drives) > 0 {
		for _, d := range o.Drives {
			args = append(args, "/drive:"+d)
		}
	} else {
		args = append(args, "/drive:Downloads,"+os.Getenv("HOME")+"/Downloads")
	}
	return args
}

// Launch spawns xfreerdp3 detached (like the bash "&"), feeding the password to
// its stdin so it never appears in the process table. Output is written straight
// to logPath (truncated fresh) so it survives this process quitting and a later
// instance can tail it. The caller should clear its copy of password once this
// returns.
func Launch(o Options, password, logPath string, onDone func(error)) error {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	cmd := exec.Command("xfreerdp3", Args(o)...)
	cmd.Stdin = strings.NewReader(password + "\n")
	cmd.Stdout = f
	cmd.Stderr = f

	if err := cmd.Start(); err != nil {
		f.Close()
		return err
	}
	f.Close() // the child holds its own dup'd fd

	go func() {
		err := cmd.Wait()
		if onDone != nil {
			onDone(err)
		}
	}()

	return nil
}
