package profile

import (
	"errors"
	"strings"

	"github.com/google/shlex"
)

type Command struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

func ParseCommand(input string) (Command, error) {
	parts, err := shlex.Split(input)
	if err != nil {
		return Command{}, err
	}
	if len(parts) == 0 {
		return Command{}, errors.New("command is empty")
	}
	for _, part := range parts {
		if isUnsafeToken(part) {
			return Command{}, errors.New("unsupported shell syntax in command")
		}
	}
	return Command{Command: parts[0], Args: parts[1:]}, nil
}

func RenderCommand(cmd Command) string {
	if len(cmd.Args) == 0 {
		return cmd.Command
	}
	return cmd.Command + " " + strings.Join(cmd.Args, " ")
}

func isUnsafeToken(token string) bool {
	switch token {
	case "sudo", "sh", "bash", "zsh", "fish", "&&", "||", "|", ">", ">>", "<", "$(", "`":
		return true
	default:
		return strings.Contains(token, "$(") || strings.Contains(token, "`") ||
			strings.Contains(token, "&&") || strings.Contains(token, "||") ||
			strings.Contains(token, "|") || strings.Contains(token, ">") || strings.Contains(token, "<")
	}
}
