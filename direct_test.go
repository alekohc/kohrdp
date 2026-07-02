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
