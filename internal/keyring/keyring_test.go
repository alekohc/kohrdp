package keyring

import "testing"

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
