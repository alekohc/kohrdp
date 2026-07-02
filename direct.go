package main

import (
	"bufio"
	"fmt"
	"os"
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
	password, err := keyring.Lookup(d.Host, d.User)
	if err != nil {
		return err
	}
	if password == "" {
		password, err = promptPassword(d.User, d.Host)
		if err != nil {
			return err
		}
		save, err := promptSavePassword()
		if err != nil {
			return err
		}
		if save {
			if err := keyring.Store(d.Host, d.User, password); err != nil {
				return err
			}
		}
	}

	ignore := d.IgnoreCert != nil && *d.IgnoreCert
	if err := rdp.Launch(rdp.Options{
		User:       d.User,
		Host:       d.Host,
		IgnoreCert: ignore,
		Multimon:   d.Multimon,
	}, password, nil, nil); err != nil {
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
