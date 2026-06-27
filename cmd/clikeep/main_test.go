package main

import (
	"os"
	"testing"
)

func TestIsTerminalReturnsFalseForRegularFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	if isTerminal(file) {
		t.Fatal("regular file reported as terminal")
	}
}

func TestResolveVersionPrefersInjectedVersion(t *testing.T) {
	got := resolveVersion("v0.2.0", "v0.1.0")
	if got != "v0.2.0" {
		t.Fatalf("version = %q, want injected version", got)
	}
}

func TestResolveVersionUsesGoInstallBuildVersion(t *testing.T) {
	got := resolveVersion("dev", "v0.1.0")
	if got != "v0.1.0" {
		t.Fatalf("version = %q, want build version", got)
	}
}

func TestResolveVersionDefaultsToDevForLocalBuild(t *testing.T) {
	got := resolveVersion("dev", "(devel)")
	if got != "dev" {
		t.Fatalf("version = %q, want dev", got)
	}
}
