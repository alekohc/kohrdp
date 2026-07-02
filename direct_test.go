package main

import "testing"

func TestParseDirectArgs(t *testing.T) {
	d, ok, err := parseDirectArgs([]string{"administrator", "192.168.11.10", "--ignore-cert", "--name", "tp-test", "--multimon"})
	if err != nil {
		t.Fatalf("parseDirectArgs() error = %v", err)
	}
	if !ok {
		t.Fatal("expected direct mode")
	}
	if d.User != "administrator" || d.Host != "192.168.11.10" || d.Name != "tp-test" || !d.Multimon {
		t.Fatalf("unexpected direct args: %#v", d)
	}
	if d.IgnoreCert == nil || !*d.IgnoreCert {
		t.Fatalf("expected ignore cert set: %#v", d)
	}
}

func TestParseDirectArgsSkipsFlagMode(t *testing.T) {
	_, ok, err := parseDirectArgs([]string{"--version"})
	if err != nil {
		t.Fatalf("parseDirectArgs() error = %v", err)
	}
	if ok {
		t.Fatal("expected non-direct mode")
	}
}

func TestParseDirectArgsErrorsOnUnknownArg(t *testing.T) {
	_, ok, err := parseDirectArgs([]string{"administrator", "192.168.11.10", "--wat"})
	if !ok {
		t.Fatal("expected direct mode")
	}
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseNamedArgs(t *testing.T) {
	name, multimon, ok, err := parseNamedArgs([]string{"win11", "--multimon"})
	if err != nil || !ok {
		t.Fatalf("parseNamedArgs() = ok %v, err %v", ok, err)
	}
	if name != "win11" || !multimon {
		t.Fatalf("unexpected named args: name=%q multimon=%v", name, multimon)
	}
}

func TestParseNamedArgsDefersToUserHost(t *testing.T) {
	// "user host" is the direct-connect form, not a name launch.
	if _, _, ok, _ := parseNamedArgs([]string{"admin", "192.168.11.10"}); ok {
		t.Fatal("expected named parsing to defer for the user/host form")
	}
	// A leading flag is not a name launch either.
	if _, _, ok, _ := parseNamedArgs([]string{"--version"}); ok {
		t.Fatal("expected named parsing to skip flag mode")
	}
}
