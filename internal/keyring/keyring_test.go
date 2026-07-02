package keyring

import (
	"testing"

	gokeyring "github.com/zalando/go-keyring"
)

func TestMungeUser(t *testing.T) {
	cases := map[string]string{
		"admin":         `.\admin`,
		"administrator": `.\administrator`,
		`DOMAIN\admin`:  `DOMAIN\admin`,
		`.\admin`:       `.\admin`,
	}
	for in, want := range cases {
		if got := MungeUser(in); got != want {
			t.Errorf("MungeUser(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEntryNameUsesMungedUser(t *testing.T) {
	if got, want := entryName("host", "admin"), `host|.\admin`; got != want {
		t.Fatalf("entryName() = %q, want %q", got, want)
	}
}

func TestLookupStoreAndClearUseGoKeyring(t *testing.T) {
	gokeyring.MockInit()
	t.Cleanup(gokeyring.MockInit)

	if err := Store("host", "admin", "pw"); err != nil {
		t.Fatalf("Store() error = %v", err)
	}

	got, err := Lookup("host", "admin")
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got != "pw" {
		t.Fatalf("Lookup() = %q, want %q", got, "pw")
	}

	if err := Clear("host", "admin"); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	got, err = Lookup("host", "admin")
	if err != nil {
		t.Fatalf("Lookup() after clear error = %v", err)
	}
	if got != "" {
		t.Fatalf("Lookup() after clear = %q, want empty", got)
	}
}
