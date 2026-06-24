package planner

import (
	"os/exec"
	"time"

	"github.com/scott9/clikeep/internal/paths"
	"github.com/scott9/clikeep/internal/profile"
)

type Options struct {
	LookupPath func(string) (string, error)
	Now        func() time.Time
}

type Plan struct {
	Items []Item `json:"items"`
}

type Item struct {
	Name         string           `json:"name"`
	Update       profile.Command  `json:"update"`
	Version      *profile.Command `json:"version,omitempty"`
	ResolvedPath string           `json:"resolved_path,omitempty"`
}

func Build(cfg profile.Config, names []string, opts Options) (Plan, []profile.Problem) {
	lookup := opts.LookupPath
	if lookup == nil {
		lookup = exec.LookPath
	}

	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameSet[name] = struct{}{}
	}

	var plan Plan
	var problems []profile.Problem
	for _, tool := range cfg.Tools {
		if len(nameSet) > 0 {
			if _, ok := nameSet[tool.Name]; !ok {
				continue
			}
		}
		if !tool.Enabled || !tool.Confirmed {
			continue
		}
		resolved, err := lookup(tool.Update.Command)
		if err != nil {
			problems = append(problems, profile.Problem{Tool: tool.Name, Severity: "error", Message: "update command not found"})
			continue
		}
		plan.Items = append(plan.Items, Item{
			Name:         tool.Name,
			Update:       tool.Update,
			Version:      tool.Version,
			ResolvedPath: resolved,
		})
	}
	return plan, problems
}

func Doctor(cfg profile.Config, p paths.Paths) []profile.Problem {
	_ = p

	problems := profile.ValidateConfig(cfg)
	_, planProblems := Build(cfg, nil, Options{})
	return append(problems, planProblems...)
}
