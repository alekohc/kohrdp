// Package rdp builds the xfreerdp3 invocation and spawns it, porting the bash
// launcher's flag set and stdin password handling verbatim.
package rdp

import (
	"os"
	"os/exec"
	"strings"

	"rdpkoh/internal/keyring"
)

// Options mirrors the bash launcher's tunable flags.
type Options struct {
	User       string // raw username; munged here for /u:
	Host       string
	WMClass    string // bare name, e.g. "work"; "" to omit
	IgnoreCert bool
	Multimon   bool
}

// Args returns the xfreerdp3 argument vector, in the same order as the bash
// launcher. Exposed for testing so the flag set can be asserted without a spawn.
func Args(o Options) []string {
	args := []string{
		"/u:" + keyring.MungeUser(o.User),
		"/from-stdin",
		"/v:" + o.Host,
	}
	if o.WMClass != "" {
		args = append(args, "/wm-class:rdpkoh-"+o.WMClass)
	}
	if o.Multimon {
		args = append(args, "/multimon")
	}
	if o.IgnoreCert {
		args = append(args, "/cert:ignore")
	}
	args = append(args,
		"+auto-reconnect", "+clipboard", "+fonts",
		"/sound", "/dynamic-resolution",
		"/gfx", "/bpp:32",
		"/drive:Downloads,"+os.Getenv("HOME")+"/Downloads",
	)
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
