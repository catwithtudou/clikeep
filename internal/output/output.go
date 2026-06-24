package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/scott9/clikeep/internal/planner"
	"github.com/scott9/clikeep/internal/profile"
)

func WritePlanText(w io.Writer, plan planner.Plan, color bool) error {
	_ = color

	if _, err := fmt.Fprintln(w, "Plan"); err != nil {
		return err
	}
	if len(plan.Items) == 0 {
		_, err := fmt.Fprintln(w, "- no eligible profiles")
		return err
	}
	for _, item := range plan.Items {
		line := fmt.Sprintf("- %s: %s", item.Name, profile.RenderCommand(item.Update))
		if item.ResolvedPath != "" {
			line += " (" + item.ResolvedPath + ")"
		}
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func WriteJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
