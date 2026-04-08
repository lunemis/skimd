package tmux

import (
	"fmt"
	"strings"
)

const (
	defaultPopupWidth  = "92%"
	defaultPopupHeight = "88%"
)

func PopupBinding(binaryPath, key string) string {
	if key == "" {
		key = "v"
	}

	command := quoteIfNeeded(binaryPath) + " ."
	command = strings.ReplaceAll(command, `"`, `\"`)

	return fmt.Sprintf(`bind %s display-popup -E -w %s -h %s -d "#{pane_current_path}" "%s"`, key, defaultPopupWidth, defaultPopupHeight, command)
}

func quoteIfNeeded(value string) string {
	if value != "" && !strings.ContainsAny(value, " \t'\"") {
		return value
	}
	return shellQuote(value)
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
