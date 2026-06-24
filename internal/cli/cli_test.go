package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scott9/clikeep/internal/profile"
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
}

func TestRunUsageListsSupportedCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(context.Background(), nil, Deps{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	for _, want := range []string{"init", "add", "list", "up", "status", "doctor"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr = %q, missing %q", stderr.String(), want)
		}
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
	if strings.Contains(stdout.String(), "lark-cli") {
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

	code := Run(context.Background(), []string{"add", "lark-cli", "--update", "lark-cli update"}, Deps{
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

	code := Run(context.Background(), []string{"add", "lark-cli", "--update", "lark-cli update", "--yes"}, Deps{
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
	if got.Name != "lark-cli" || !got.Enabled || !got.Confirmed ||
		profile.RenderCommand(got.Update) != "lark-cli update" {
		t.Fatalf("profile = %#v, want confirmed enabled lark-cli update", got)
	}

	stdout.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"list"}, Deps{
		Stdout:     &stdout,
		Stderr:     &stderr,
		ConfigHome: configFile,
		StateHome:  filepath.Join(dir, "state"),
	})
	if code != 0 || !strings.Contains(stdout.String(), "lark-cli") {
		t.Fatalf("list failed or missing profile, code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
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

	if code := Run(context.Background(), []string{"add", "lark-cli", "--update", "lark-cli update", "--yes"}, deps); code != 0 {
		t.Fatalf("first add exit code = %d", code)
	}
	if code := Run(context.Background(), []string{"add", "lark-cli", "--update", "lark-cli update", "--yes"}, deps); code == 0 {
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
	if !strings.Contains(stderr.String(), "update command not found") {
		t.Fatalf("stderr = %q, want missing command problem", stderr.String())
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
	for _, want := range []string{"Plan", "Run", "Summary", "echo", "success"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output = %q, missing %q", out, want)
		}
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
	for _, want := range []string{"echo", "enabled=true", "confirmed=true", "success", ".log"} {
		if !strings.Contains(out, want) {
			t.Fatalf("status output = %q, missing %q", out, want)
		}
	}
}
