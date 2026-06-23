package profile

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Tools []Profile `toml:"tools"`
}

type Profile struct {
	Name      string   `toml:"name"`
	Enabled   bool     `toml:"enabled"`
	Confirmed bool     `toml:"confirmed"`
	Aliases   []string `toml:"aliases,omitempty"`
	Tags      []string `toml:"tags,omitempty"`
	Update    Command  `toml:"update"`
	Version   *Command `toml:"version,omitempty"`
}

type Problem struct {
	Tool     string
	Severity string
	Message  string
}

func Load(path string) (Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ValidateConfig(cfg Config) []Problem {
	var problems []Problem
	seen := make(map[string]struct{}, len(cfg.Tools))

	for _, tool := range cfg.Tools {
		if tool.Name == "" {
			problems = append(problems, Problem{Severity: "error", Message: "profile name is required"})
			continue
		}
		if _, ok := seen[tool.Name]; ok {
			problems = append(problems, Problem{Tool: tool.Name, Severity: "error", Message: "profile name must be unique"})
		} else {
			seen[tool.Name] = struct{}{}
		}
		if tool.Update.Command == "" {
			problems = append(problems, Problem{Tool: tool.Name, Severity: "error", Message: "update command is required"})
		}
		problems = append(problems, validateCommand(tool.Name, "update", tool.Update)...)
		if tool.Version != nil {
			problems = append(problems, validateCommand(tool.Name, "version", *tool.Version)...)
		}
	}

	return problems
}

func validateCommand(tool, field string, cmd Command) []Problem {
	if cmd.Command == "" {
		return []Problem{{Tool: tool, Severity: "error", Message: field + " command is required"}}
	}
	if isUnsafeToken(cmd.Command) {
		return []Problem{{Tool: tool, Severity: "error", Message: field + " command is unsafe"}}
	}
	for _, arg := range cmd.Args {
		if isUnsafeToken(arg) {
			return []Problem{{Tool: tool, Severity: "error", Message: field + " args contain unsupported shell syntax"}}
		}
	}
	return nil
}
