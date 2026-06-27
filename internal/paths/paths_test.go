package paths

import (
	"testing"
)

func TestDefaultUsesXDGPaths(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/config")
	t.Setenv("XDG_STATE_HOME", "/tmp/state")

	got := Default()
	want := Paths{
		ConfigFile: "/tmp/config/clikeep/config.toml",
		StateDir:   "/tmp/state/clikeep",
		RunsDir:    "/tmp/state/clikeep/runs",
		LatestFile: "/tmp/state/clikeep/latest-run",
	}

	if got != want {
		t.Fatalf("Default() = %#v, want %#v", got, want)
	}
}

func TestDefaultFallsBackToHome(t *testing.T) {
	t.Setenv("HOME", "/home/tester")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	got := Default()
	want := Paths{
		ConfigFile: "/home/tester/.config/clikeep/config.toml",
		StateDir:   "/home/tester/.local/state/clikeep",
		RunsDir:    "/home/tester/.local/state/clikeep/runs",
		LatestFile: "/home/tester/.local/state/clikeep/latest-run",
	}

	if got != want {
		t.Fatalf("Default() = %#v, want %#v", got, want)
	}
}

func TestFromHomesBuildsExactPaths(t *testing.T) {
	got := FromHomes("/config", "/state")
	want := Paths{
		ConfigFile: "/config/clikeep/config.toml",
		StateDir:   "/state/clikeep",
		RunsDir:    "/state/clikeep/runs",
		LatestFile: "/state/clikeep/latest-run",
	}

	if got != want {
		t.Fatalf("FromHomes() = %#v, want %#v", got, want)
	}
}
