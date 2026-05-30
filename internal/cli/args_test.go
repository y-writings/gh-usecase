package cli

import (
	"reflect"
	"testing"
)

func TestParseArgsParsesOptionsHelpAndPositionals(t *testing.T) {
	parsed := ParseArgs([]string{
		"pr-list",
		"--owner", "octo",
		"--repo=hello",
		"--draft",
		"--unknown", "kept",
		"--help",
	})

	wantOptions := map[string]string{
		"owner":   "octo",
		"repo":    "hello",
		"unknown": "kept",
	}
	if !reflect.DeepEqual(parsed.Options, wantOptions) {
		t.Fatalf("Options = %#v, want %#v", parsed.Options, wantOptions)
	}

	wantPositionals := []string{"pr-list"}
	if !reflect.DeepEqual(parsed.Positionals, wantPositionals) {
		t.Fatalf("Positionals = %#v, want %#v", parsed.Positionals, wantPositionals)
	}

	if !parsed.Help {
		t.Fatal("Help = false, want true")
	}
}

func TestParseArgsRecognizesShortHelp(t *testing.T) {
	parsed := ParseArgs([]string{"-h"})

	if !parsed.Help {
		t.Fatal("Help = false, want true")
	}
}

func TestParseArgsDoesNotAddMissingOptionValue(t *testing.T) {
	parsed := ParseArgs([]string{"--owner"})

	if _, ok := parsed.Options["owner"]; ok {
		t.Fatalf("Options[owner] = %q, want missing", parsed.Options["owner"])
	}
}
