// Package keyring wraps secret-tool, matching the bash launcher's munge and
// attribute layout exactly so passwords stored by either tool resolve.
package keyring

import (
	"os/exec"
	"strings"
)

// MungeUser replicates the bash launcher: if the username has no backslash,
// prefix ".\". Keyring entries were stored against the munged name, so every
// lookup/store/clear must munge first.
func MungeUser(user string) string {
	if !strings.Contains(user, `\`) {
		return `.\` + user
	}
	return user
}

// Lookup returns the stored password for host/user, or "" if none. The user is
// munged here; callers pass the raw username.
func Lookup(host, user string) (string, error) {
	out, err := exec.Command("secret-tool", "lookup",
		"rdp", "host", host, "rdp", "user", MungeUser(user)).Output()
	if err != nil {
		// secret-tool exits non-zero when the secret is absent; treat that as
		// "no password" rather than a hard error.
		if _, ok := err.(*exec.ExitError); ok {
			return "", nil
		}
		return "", err
	}
	// Bash captures the password via $(...), which strips trailing newlines.
	// Trim here so a secret stored by either tool resolves to the same bytes.
	return strings.TrimRight(string(out), "\n"), nil
}

// Store saves the password under the same label and attributes the bash
// launcher uses. The password is fed via stdin, never the command line.
func Store(host, user, password string) error {
	cmd := exec.Command("secret-tool", "store", "--label=RDP "+host,
		"rdp", "host", host, "rdp", "user", MungeUser(user))
	cmd.Stdin = strings.NewReader(password)
	return cmd.Run()
}

// Clear removes the stored password, matching --reprompt in the bash launcher.
func Clear(host, user string) error {
	err := exec.Command("secret-tool", "clear",
		"rdp", "host", host, "rdp", "user", MungeUser(user)).Run()
	if _, ok := err.(*exec.ExitError); ok {
		return nil
	}
	return err
}
