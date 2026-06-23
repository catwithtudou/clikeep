package paths

import (
	"os"
	"path/filepath"
)

type Paths struct {
	ConfigFile string
	StateDir   string
	RunsDir    string
	LatestFile string
}

func Default() Paths {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}

	configHome := getenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	stateHome := getenv("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	return FromHomes(configHome, stateHome)
}

func FromHomes(configHome, stateHome string) Paths {
	stateDir := filepath.Join(stateHome, "clikeep")
	return Paths{
		ConfigFile: filepath.Join(configHome, "clikeep", "config.toml"),
		StateDir:   stateDir,
		RunsDir:    filepath.Join(stateDir, "runs"),
		LatestFile: filepath.Join(stateDir, "latest-run"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
