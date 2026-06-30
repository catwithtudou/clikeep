package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/catwithtudou/clikeep/internal/planner"
	"github.com/catwithtudou/clikeep/internal/profile"
)

func WritePlanText(w io.Writer, plan planner.Plan, color bool) error {
	style := NewStyle(color)

	if _, err := fmt.Fprintln(w, style.Heading("Plan")); err != nil {
		return err
	}
	if len(plan.Items) == 0 {
		_, err := fmt.Fprintln(w, "  no eligible profiles")
		return err
	}
	for i, item := range plan.Items {
		if _, err := fmt.Fprintf(w, "  %d. %s\n", i+1, style.Accent(item.Name)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "     command: %s\n", profile.RenderCommand(item.Update)); err != nil {
			return err
		}
		if item.ResolvedPath != "" {
			if _, err := fmt.Fprintf(w, "     path:    %s\n", style.Dim(item.ResolvedPath)); err != nil {
				return err
			}
		}
	}
	return nil
}

type Style struct {
	Enabled bool
}

func NewStyle(enabled bool) Style {
	return Style{Enabled: enabled}
}

func (s Style) Heading(text string) string {
	return s.wrap("1;36", text)
}

func (s Style) Dim(text string) string {
	return s.wrap("2", text)
}

func (s Style) Accent(text string) string {
	return s.wrap("36", text)
}

func (s Style) Success(text string) string {
	return s.wrap("32", text)
}

func (s Style) Warning(text string) string {
	return s.wrap("33", text)
}

func (s Style) Error(text string) string {
	return s.wrap("31", text)
}

func (s Style) Status(status string) string {
	switch status {
	case "success":
		return s.Success(status)
	case "failed":
		return s.Error(status)
	case "skipped":
		return s.Warning(status)
	case "running":
		return s.Accent(status)
	default:
		return status
	}
}

func (s Style) wrap(code, text string) string {
	if !s.Enabled || text == "" {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func ProgressLine(style Style, current, total int, name, status string) string {
	return fmt.Sprintf("  %-7s [%d/%d] %s", style.Status(status), current, total, name)
}

func WriteJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
