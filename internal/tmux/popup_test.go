package tmux

import (
	"strings"
	"testing"
)

func TestPopupBindingContainsExpectedPieces(t *testing.T) {
	got := PopupBinding("/home/user/go/bin/skimd", "v")

	required := []string{
		"bind v display-popup",
		"-w 92%",
		"-h 88%",
		`-d "#{pane_current_path}"`,
		"/home/user/go/bin/skimd",
		" .",
	}

	for _, part := range required {
		if !strings.Contains(got, part) {
			t.Fatalf("expected binding %q to contain %q", got, part)
		}
	}
}

func TestPopupBindingQuotesPathsWithSpaces(t *testing.T) {
	got := PopupBinding("/tmp/my tools/skimd", "v")
	if !strings.Contains(got, `'/tmp/my tools/skimd' .`) {
		t.Fatalf("expected quoted path in binding, got %q", got)
	}
}
