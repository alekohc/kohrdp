package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"golang.org/x/term"

	"rdpkoh/internal/config"
	"rdpkoh/internal/keyring"
	"rdpkoh/internal/rdp"
)

type directArgs struct {
	User       string
	Host       string
	Name       string
	IgnoreCert *bool
	Multimon   bool
}

func parseDirectArgs(args []string) (directArgs, bool, error) {
	if len(args) < 2 || strings.HasPrefix(args[0], "-") {
		return directArgs{}, false, nil
	}

	d := directArgs{User: args[0], Host: args[1]}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--ignore-cert":
			v := true
			d.IgnoreCert = &v
		case "--multimon":
			d.Multimon = true
		case "--name":
			i++
			if i >= len(args) {
				return directArgs{}, true, fmt.Errorf("--name requires a value")
			}
			d.Name = args[i]
		default:
			return directArgs{}, true, fmt.Errorf("unknown direct-connect argument %q", args[i])
		}
	}

	return d, true, nil
}

func runDirectConnect(d directArgs) error {
	password, err := ensurePassword(d.User, d.Host)
	if err != nil {
		return err
	}

	logPath := os.DevNull
	if d.Name != "" {
		if err := os.MkdirAll(rdp.LogDir(), 0o700); err == nil {
			logPath = rdp.LogPath(d.Name)
		}
	}
	s := config.Session{User: d.User, Host: d.Host, IgnoreCert: d.IgnoreCert}
	if err := rdp.Launch(rdp.OptionsFromSession(d.User, d.Host, s, d.Multimon), password, logPath, nil); err != nil {
		password = ""
		return err
	}
	password = ""

	if d.Name != "" {
		if err := saveDirectSession(d); err != nil {
			return err
		}
	}

	return nil
}

// parseNamedArgs recognises "rdpkoh <name> [--multimon]": a single non-flag
// argument (optionally followed by flags) that names a saved session to launch.
// It defers (ok=false) to direct-connect parsing when a second positional
// argument is present (the user/host form).
func parseNamedArgs(args []string) (name string, multimon, ok bool, err error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return "", false, false, nil
	}
	if len(args) >= 2 && !strings.HasPrefix(args[1], "-") {
		return "", false, false, nil
	}
	name = args[0]
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--multimon":
			multimon = true
		default:
			return "", false, true, fmt.Errorf("unknown argument %q", args[i])
		}
	}
	return name, multimon, true, nil
}

// runNamedConnect launches a saved session by name, using its stored properties.
func runNamedConnect(name string, multimon bool) error {
	sessions, err := config.Load()
	if err != nil {
		return err
	}
	s, ok := sessions[name]
	if !ok {
		names := make([]string, 0, len(sessions))
		for n := range sessions {
			names = append(names, n)
		}
		sort.Strings(names)
		return fmt.Errorf("no saved session named %q; available: %s", name, strings.Join(names, ", "))
	}

	password, err := ensurePassword(s.User, s.Host)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(rdp.LogDir(), 0o700)
	if err := rdp.Launch(rdp.OptionsFromSession(s.User, s.Host, s, multimon), password, rdp.LogPath(name), nil); err != nil {
		password = ""
		return err
	}
	password = ""

	s.LastUsed = time.Now().Unix()
	sessions[name] = s
	return config.Save(sessions)
}

// ensurePassword returns the keyring password for user@host, prompting (and
// optionally saving) when it is missing.
func ensurePassword(user, host string) (string, error) {
	password, err := keyring.Lookup(host, user)
	if err != nil {
		return "", err
	}
	if password != "" {
		return password, nil
	}
	password, err = promptPassword(user, host)
	if err != nil {
		return "", err
	}
	save, err := promptSavePassword()
	if err != nil {
		return "", err
	}
	if save {
		if err := keyring.Store(host, user, password); err != nil {
			return "", err
		}
	}
	return password, nil
}

func saveDirectSession(d directArgs) error {
	sessions, err := config.Load()
	if err != nil {
		return err
	}

	s := sessions[d.Name]
	s.User = d.User
	s.Host = d.Host
	if d.IgnoreCert != nil {
		s.IgnoreCert = d.IgnoreCert
	}
	s.LastUsed = time.Now().Unix()
	sessions[d.Name] = s
	return config.Save(sessions)
}

func promptPassword(user, host string) (string, error) {
	fmt.Fprintf(os.Stderr, "Password for %s@%s: ", user, host)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

func promptSavePassword() (bool, error) {
	fmt.Fprint(os.Stderr, "Save password to keyring? [y/N] ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(line)
	return line == "y" || line == "Y", nil
}
