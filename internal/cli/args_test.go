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

func TestParseArgsTracksOptionOccurrences(t *testing.T) {
	parsed := ParseArgs([]string{
		"codeql-default-setup",
		"--owner", "octo",
		"--repo=hello",
		"--languages", "go",
		"--languages", "python",
		"--flag-without-value",
	})

	wantOccurrences := map[string]int{
		"owner":              1,
		"repo":               1,
		"languages":          2,
		"flag-without-value": 1,
	}
	if !reflect.DeepEqual(parsed.OptionOccurrences, wantOccurrences) {
		t.Fatalf("OptionOccurrences = %#v, want %#v", parsed.OptionOccurrences, wantOccurrences)
	}

	if parsed.Options["languages"] != "python" {
		t.Fatalf("Options[languages] = %q, want last value python", parsed.Options["languages"])
	}
	if _, ok := parsed.Options["flag-without-value"]; ok {
		t.Fatalf("Options[flag-without-value] = %q, want missing", parsed.Options["flag-without-value"])
	}
}

func TestParseArgsTracksMissingOptionValueOccurrence(t *testing.T) {
	parsed := ParseArgs([]string{"codeql-default-setup", "--owner"})

	if parsed.OptionOccurrences["owner"] != 1 {
		t.Fatalf("OptionOccurrences[owner] = %d, want 1", parsed.OptionOccurrences["owner"])
	}
	if _, ok := parsed.Options["owner"]; ok {
		t.Fatalf("Options[owner] = %q, want missing", parsed.Options["owner"])
	}
}
