package runlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Summary struct {
	RunID     string    `json:"run_id"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Results   []Result  `json:"results"`
}

type Result struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	ExitCode int    `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
	LogPath  string `json:"log_path"`
}

func WriteSummary(stateDir string, summary Summary) error {
	runDir := filepath.Join(stateDir, "runs", summary.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runDir, "run-summary.json"), data, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(stateDir, "latest-run"), []byte(summary.RunID), 0o644)
}

func ReadLatest(stateDir string) (Summary, bool, error) {
	var summary Summary
	data, err := os.ReadFile(filepath.Join(stateDir, "latest-run"))
	if err != nil {
		if os.IsNotExist(err) {
			return summary, false, nil
		}
		return summary, false, err
	}
	runID := strings.TrimSpace(string(data))
	if runID == "" {
		return summary, false, nil
	}
	summaryData, err := os.ReadFile(filepath.Join(stateDir, "runs", runID, "run-summary.json"))
	if err != nil {
		return summary, false, err
	}
	if err := json.Unmarshal(summaryData, &summary); err != nil {
		return summary, false, err
	}
	return summary, true, nil
}
