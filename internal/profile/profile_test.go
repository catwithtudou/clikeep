package profile

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestValidateMinimumProfile(t *testing.T) {
	cfg := Config{Tools: []Profile{{
		Name:      "demo-cli",
		Enabled:   true,
		Confirmed: true,
		Update:    Command{Command: "demo-cli", Args: []string{"update"}},
	}}}

	problems := ValidateConfig(cfg)
	if len(problems) != 0 {
		t.Fatalf("problems = %#v, want none", problems)
	}
}

func TestValidateDuplicateNames(t *testing.T) {
	cfg := Config{Tools: []Profile{
		{
			Name:      "demo-cli",
			Enabled:   true,
			Confirmed: true,
			Update:    Command{Command: "demo-cli", Args: []string{"update"}},
		},
		{
			Name:      "demo-cli",
			Enabled:   true,
			Confirmed: true,
			Update:    Command{Command: "demo-cli", Args: []string{"update"}},
		},
	}}

	problems := ValidateConfig(cfg)
	if len(problems) == 0 {
		t.Fatal("expected duplicate-name problem")
	}
}

func TestValidateOptionalVersionCommand(t *testing.T) {
	cfg := Config{Tools: []Profile{{
		Name:      "demo-cli",
		Enabled:   true,
		Confirmed: true,
		Update:    Command{Command: "demo-cli", Args: []string{"update"}},
		Version:   &Command{Command: "demo-cli", Args: []string{"--version"}},
	}}}

	problems := ValidateConfig(cfg)
	if len(problems) != 0 {
		t.Fatalf("problems = %#v, want none", problems)
	}
}

func TestValidateRequiresUpdateArgs(t *testing.T) {
	cfg := Config{Tools: []Profile{{
		Name:      "demo-cli",
		Enabled:   true,
		Confirmed: true,
		Update:    Command{Command: "demo-cli"},
	}}}

	problems := ValidateConfig(cfg)
	if len(problems) == 0 {
		t.Fatal("expected update args problem")
	}
}

func TestValidateAllowsStructuredArgLiterals(t *testing.T) {
	cfg := Config{Tools: []Profile{{
		Name:      "literal-cli",
		Enabled:   true,
		Confirmed: true,
		Update: Command{
			Command: "literal-cli",
			Args: []string{
				"value|literal",
				"a>b",
				"x&&y",
				"$(literal)",
				"`literal`",
			},
		},
	}}}

	problems := ValidateConfig(cfg)
	if len(problems) != 0 {
		t.Fatalf("problems = %#v, want none", problems)
	}
}

func TestValidateRejectsProfileEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	data := []byte(`
[[tools]]
name = "demo-cli"
enabled = true
confirmed = true

[tools.update]
command = "demo-cli"
args = ["update"]

[tools.env]
FOO = "bar"
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	problems := ValidateConfig(cfg)
	if len(problems) == 0 {
		t.Fatal("expected profile env problem")
	}
}

func TestParseCommandRejectsShellComposition(t *testing.T) {
	bad := []string{
		"demo-cli update && echo done",
		"demo-cli update | tee out",
		"demo-cli update > out",
		"demo-cli update < in",
		"demo-cli update $(echo done)",
		"demo-cli update `echo done`",
		"sudo demo-cli update",
		"sh -c 'demo-cli update'",
		"bash -c 'demo-cli update'",
	}

	for _, input := range bad {
		if _, err := ParseCommand(input); err == nil {
			t.Fatalf("ParseCommand(%q) succeeded, want error", input)
		}
	}
}

func TestLoadAndSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	want := Config{Tools: []Profile{{
		Name:      "demo-cli",
		Enabled:   true,
		Confirmed: true,
		Aliases:   []string{"demo"},
		Tags:      []string{"sample"},
		Update:    Command{Command: "demo-cli", Args: []string{"update"}},
		Version:   &Command{Command: "demo-cli", Args: []string{"--version"}},
	}}}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got.Tools) != len(want.Tools) {
		t.Fatalf("len(got.Tools) = %d, want %d", len(got.Tools), len(want.Tools))
	}
	if got.Tools[0].Name != want.Tools[0].Name ||
		got.Tools[0].Enabled != want.Tools[0].Enabled ||
		got.Tools[0].Confirmed != want.Tools[0].Confirmed ||
		!slices.Equal(got.Tools[0].Aliases, want.Tools[0].Aliases) ||
		!slices.Equal(got.Tools[0].Tags, want.Tools[0].Tags) ||
		RenderCommand(got.Tools[0].Update) != RenderCommand(want.Tools[0].Update) ||
		got.Tools[0].Version == nil ||
		RenderCommand(*got.Tools[0].Version) != RenderCommand(*want.Tools[0].Version) {
		t.Fatalf("got = %#v, want %#v", got, want)
	}
}

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Tools) != 0 {
		t.Fatalf("got = %#v, want empty config", got)
	}
}
