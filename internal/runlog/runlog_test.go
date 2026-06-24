package runlog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSummaryWritesSummaryAndLatestPointer(t *testing.T) {
	dir := t.TempDir()
	summary := Summary{
		RunID: "2026-06-23T21-00-00",
		Results: []Result{{
			Name:    "lark-cli",
			Status:  "success",
			LogPath: filepath.Join(dir, "runs", "2026-06-23T21-00-00", "lark-cli.log"),
		}},
	}

	if err := WriteSummary(dir, summary); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "runs", summary.RunID, "run-summary.json")); err != nil {
		t.Fatal(err)
	}
	latest, err := os.ReadFile(filepath.Join(dir, "latest-run"))
	if err != nil {
		t.Fatal(err)
	}
	if string(latest) != summary.RunID {
		t.Fatalf("latest = %q, want %q", latest, summary.RunID)
	}
}
