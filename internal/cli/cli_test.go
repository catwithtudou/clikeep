package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/catwithtudou/clikeep/internal/profile"
)

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"unknown"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown command", stderr.String())
	}
	if !strings.Contains(stderr.String(), "clikeep help") {
		t.Fatalf("stderr = %q, want help guidance", stderr.String())
	}
}

func TestRunWithoutArgsPrintsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), nil, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	for _, want := range []string{"clikeep - local-first update manager", "Usage:", "init", "add", "list", "up", "update", "status", "doctor", "self-update", "version"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestHelpFlagPrintsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--help"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Commands:") || !strings.Contains(stdout.String(), "self-update") {
		t.Fatalf("stdout = %q, want help", stdout.String())
	}
}

func TestVersionCommandUsesInjectedVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"version"}, Deps{
		Stdout:  &stdout,
		Stderr:  &stderr,
		Version: "v0.1.0",
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if got, want := stdout.String(), "clikeep v0.1.0\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestVersionFlagDefaultsToDev(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"--version"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if got, want := stdout.String(), "clikeep dev\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestSelfUpdateDryRunPrintsGoInstallCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	called := false
	code := Run(context.Background(), []string{"self-update", "--dry-run", "--version", "v0.1.0"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
		SelfUpdateRunner: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			called = true
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if called {
		t.Fatal("dry-run executed self-update runner")
	}
	for _, want := range []string{"Self-update", "go install github.com/catwithtudou/clikeep/cmd/clikeep@v0.1.0"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
}

func TestSelfUpdateWithYesRunsGoInstallLatest(t *testing.T) {
	var stdout, stderr bytes.Buffer
	var gotArgs []string
	code := Run(context.Background(), []string{"self-update", "--yes"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
		SelfUpdateRunner: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	wantArgs := []string{"install", "github.com/catwithtudou/clikeep/cmd/clikeep@latest"}
	if strings.Join(gotArgs, " ") != strings.Join(wantArgs, " ") {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	if !strings.Contains(stdout.String(), "updated clikeep") {
		t.Fatalf("stdout = %q, want success message", stdout.String())
	}
}

func TestSelfUpdateRequiresYesWhenNonInteractive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"self-update"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
		SelfUpdateRunner: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			t.Fatal("self-update runner should not run without confirmation")
			return nil
		},
	})
	if code == 0 {
		t.Fatal("self-update without --yes in non-interactive mode succeeded, want failure")
	}
	if !strings.Contains(stderr.String(), "--yes") {
		t.Fatalf("stderr = %q, want --yes guidance", stderr.String())
	}
}

func TestSelfUpdateReportsRunnerFailure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"self-update", "--yes"}, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
		SelfUpdateRunner: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			return errors.New("go install failed")
		},
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "self-update failed") {
		t.Fatalf("stderr = %q, want failure", stderr.String())
	}
}

func TestInitCreatesEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	stateDir := filepath.Join(dir, "state")
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"init"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  stateDir,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "demo-cli") {
		t.Fatalf("init output contains example profile: %s", stdout.String())
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("config file = %q, want empty file", string(data))
	}
	if _, err := os.Stat(filepath.Join(stateDir, "runs")); err != nil {
		t.Fatalf("runs dir missing: %v", err)
	}
}

func TestAddRequiresYesWhenNonInteractive(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"add", "demo-cli", "--update", "demo-cli update"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: filepath.Join(dir, "config.toml"),
		StateHome:  filepath.Join(dir, "state"),
	})
	if code == 0 {
		t.Fatal("add without --yes in non-interactive mode succeeded, want failure")
	}
}

func TestAddWithYesWritesConfirmedEnabledProfile(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), []string{"add", "demo-cli", "--update", "demo-cli update", "--yes"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	cfg, err := profile.Load(configFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Tools) != 1 {
		t.Fatalf("profiles = %#v, want one profile", cfg.Tools)
	}
	got := cfg.Tools[0]
	if got.Name != "demo-cli" || !got.Enabled || !got.Confirmed ||
		profile.RenderCommand(got.Update) != "demo-cli update" {
		t.Fatalf("profile = %#v, want confirmed enabled demo-cli update", got)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"list"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	})
	if code != 0 {
		t.Fatalf("list failed or missing profile, code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"Profiles", "NAME", "ENABLED", "CONFIRMED", "UPDATE", "demo-cli", "true", "demo-cli update"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("list output = %q, missing %q", stdout.String(), want)
		}
	}
	if strings.Contains(stdout.String(), "enabled=true") {
		t.Fatalf("list output = %q, want table output instead of key-value rows", stdout.String())
	}
}

func TestAddRejectsDuplicateProfileName(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	deps := Deps{
		Stdout:     &bytes.Buffer{},
		Stderr:     &bytes.Buffer{},
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	}

	if code := Run(context.Background(), []string{"add", "demo-cli", "--update", "demo-cli update", "--yes"}, deps); code != 0 {
		t.Fatalf("first add exit code = %d", code)
	}
	if code := Run(context.Background(), []string{"add", "demo-cli", "--update", "demo-cli update", "--yes"}, deps); code == 0 {
		t.Fatal("duplicate add succeeded, want failure")
	}
}

func TestDoctorReportsMissingUpdateCommand(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "missing",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "clikeep-definitely-missing-cli", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"doctor"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	for _, want := range []string{"Doctor", "Config", "Commands", "Latest Run", "update command not found"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, missing %q", stdout.String(), want)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want grouped doctor output on stdout", stderr.String())
	}
}

func TestUpDryRunJSONDoesNotCreateRunDir(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	stateDir := filepath.Join(dir, "state")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"up", "--dry-run", "--json"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  stateDir,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		Items []struct {
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output %q: %v", stdout.String(), err)
	}
	if len(decoded.Items) != 1 || decoded.Items[0].Name != "echo" {
		t.Fatalf("json output = %s, want echo plan item", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(stateDir, "runs")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created runs dir or unexpected stat error: %v", err)
	}
}

func TestUpdateDryRunColorControls(t *testing.T) {
	cases := []struct {
		name        string
		args        []string
		stdoutIsTTY bool
		noColorEnv  bool
		wantANSI    bool
	}{
		{name: "tty", args: []string{"update", "--dry-run"}, stdoutIsTTY: true, wantANSI: true},
		{name: "non tty", args: []string{"update", "--dry-run"}, stdoutIsTTY: false, wantANSI: false},
		{name: "no color flag", args: []string{"update", "--dry-run", "--no-color"}, stdoutIsTTY: true, wantANSI: false},
		{name: "no color env", args: []string{"update", "--dry-run"}, stdoutIsTTY: true, noColorEnv: true, wantANSI: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			configFile := filepath.Join(dir, "config.toml")
			cfg := profile.Config{Tools: []profile.Profile{{
				Name:      "echo",
				Enabled:   true,
				Confirmed: true,
				Update:    profile.Command{Command: "echo", Args: []string{"update"}},
			}}}
			if err := profile.Save(configFile, cfg); err != nil {
				t.Fatalf("Save() error = %v", err)
			}
			if tc.noColorEnv {
				t.Setenv("NO_COLOR", "1")
			} else {
				t.Setenv("NO_COLOR", "")
				if err := os.Unsetenv("NO_COLOR"); err != nil {
					t.Fatalf("Unsetenv() error = %v", err)
				}
			}

			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), tc.args, Deps{
				Stdout:      &stdout,
				Stderr:      &stderr,
				ConfigHome:  configFile,
				StateHome:   filepath.Join(dir, "state"),
				StdoutIsTTY: tc.stdoutIsTTY,
			})
			if code != 0 {
				t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
			}
			hasANSI := strings.Contains(stdout.String(), "\x1b[")
			if hasANSI != tc.wantANSI {
				t.Fatalf("stdout = %q, hasANSI=%t want %t", stdout.String(), hasANSI, tc.wantANSI)
			}
		})
	}
}

func TestListNoColorDisablesColorWhenTTY(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"list", "--no-color"}, Deps{
		Stdout:      &stdout,
		Stderr:      &stderr,
		ConfigHome:  configFile,
		StateHome:   filepath.Join(dir, "state"),
		StdoutIsTTY: true,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "\x1b[") {
		t.Fatalf("list output contains ANSI despite --no-color: %q", stdout.String())
	}
}

func TestUpdateJSONDisablesColorEvenWhenTTY(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	stateDir := filepath.Join(dir, "state")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"update", "--yes", "--json"}, Deps{
		Stdout:      &stdout,
		Stderr:      &stderr,
		ConfigHome:  configFile,
		StateHome:   stateDir,
		StdoutIsTTY: true,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "\x1b[") {
		t.Fatalf("json output contains ANSI: %q", stdout.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output %q: %v", stdout.String(), err)
	}
}

func TestUpdateAliasDryRunUsesUpBehavior(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"update", "--dry-run"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Plan") || !strings.Contains(stdout.String(), "echo") {
		t.Fatalf("stdout = %q, want dry-run plan", stdout.String())
	}
}

func TestUpRequiresYesWhenNonInteractive(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"up"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	})
	if code == 0 {
		t.Fatal("up without --yes in non-interactive mode succeeded, want failure")
	}
	if !strings.Contains(stderr.String(), "--yes") {
		t.Fatalf("stderr = %q, want --yes guidance", stderr.String())
	}
}

func TestUpWithYesJSONWritesRunSummary(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	stateDir := filepath.Join(dir, "state")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"up", "--yes", "--json"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  stateDir,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	var decoded struct {
		RunID   string `json:"run_id"`
		Results []struct {
			Name    string `json:"name"`
			Status  string `json:"status"`
			LogPath string `json:"log_path"`
		} `json:"results"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json output %q: %v", stdout.String(), err)
	}
	if decoded.RunID == "" || len(decoded.Results) != 1 || decoded.Results[0].Status != "success" {
		t.Fatalf("summary = %#v, want one successful result", decoded)
	}
	latest, err := os.ReadFile(filepath.Join(stateDir, "latest-run"))
	if err != nil {
		t.Fatalf("latest-run missing: %v", err)
	}
	if string(latest) != decoded.RunID {
		t.Fatalf("latest-run = %q, want %q", latest, decoded.RunID)
	}
	if _, err := os.Stat(decoded.Results[0].LogPath); err != nil {
		t.Fatalf("log path missing: %v", err)
	}
}

func TestUpWithYesTextUsesStagedOutput(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), []string{"up", "--yes"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	})
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"Plan",
		"  1. echo",
		"     command: echo update",
		"Run",
		"  [1/1] echo  [##########] running",
		"  [1/1] echo  [##########] success",
		"Summary",
		"  success: 1  failed: 0  skipped: 0",
		"  success  echo",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, missing %q", out, want)
		}
	}
	if strings.Contains(out, ".log") {
		t.Fatalf("output = %q, summary should not show log paths for successful runs", out)
	}
}

func TestStatusShowsConfiguredProfileAndLatestResult(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.toml")
	stateDir := filepath.Join(dir, "state")
	cfg := profile.Config{Tools: []profile.Profile{{
		Name:      "echo",
		Enabled:   true,
		Confirmed: true,
		Update:    profile.Command{Command: "echo", Args: []string{"update"}},
	}}}
	if err := profile.Save(configFile, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var stdout, stderr bytes.Buffer
	deps := Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  stateDir,
	}
	if code := Run(context.Background(), []string{"up", "--yes", "--json"}, deps); code != 0 {
		t.Fatalf("up exit code = %d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := Run(context.Background(), []string{"status"}, deps)
	if code != 0 {
		t.Fatalf("status exit code = %d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"Status", "NAME", "ENABLED", "CONFIRMED", "LATEST", "LOG", "echo", "true", "success", ".log"} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output = %q, missing %q", out, want)
		}
	}
	if strings.Contains(out, "enabled=true") {
		t.Fatalf("status output = %q, want table output instead of key-value rows", out)
	}
}
