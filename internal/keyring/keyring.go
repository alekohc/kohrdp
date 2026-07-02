// Package keyring wraps the OS keyring.
package keyring

import (
	"errors"
	"strings"

	gokeyring "github.com/zalando/go-keyring"
)

const service = "kokoarch-rdp"

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
	password, err := gokeyring.Get(service, entryName(host, user))
	if err == nil {
		return password, nil
	}
	if !errors.Is(err, gokeyring.ErrNotFound) {
		return "", err
	}
	return "", nil
}

// Store saves the password in the OS keyring.
func Store(host, user, password string) error {
	return gokeyring.Set(service, entryName(host, user), password)
}

// Clear removes the stored password.
func Clear(host, user string) error {
	err := gokeyring.Delete(service, entryName(host, user))
	if err != nil && !errors.Is(err, gokeyring.ErrNotFound) {
		return err
	}
	return nil
}

func entryName(host, user string) string {
	return host + "|" + MungeUser(user)
}
