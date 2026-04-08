package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func padOrTruncate(s string, width int) string {
	if width <= 0 {
		return ""
	}

	visible := ansi.StringWidth(s)
	if visible > width {
		return ansi.Truncate(s, width, "")
	}
	if visible < width {
		return s + strings.Repeat(" ", width-visible)
	}
	return s
}

func fixedBox(content string, width, height int) string {
	if height <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	result := make([]string, height)
	for i := 0; i < height; i++ {
		if i < len(lines) {
			result[i] = padOrTruncate(lines[i], width)
			continue
		}
		result[i] = strings.Repeat(" ", max(width, 0))
	}
	return strings.Join(result, "\n")
}

func joinHorizontalFixed(left, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	lineCount := max(len(leftLines), len(rightLines))

	result := make([]string, lineCount)
	for i := 0; i < lineCount; i++ {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		result[i] = l + r
	}

	return strings.Join(result, "\n")
}

func drawBorder(content string, width, height int) string {
	if width < 2 {
		return ""
	}
	innerWidth := width - 2
	lines := strings.Split(content, "\n")
	result := make([]string, 0, height+2)

	result = append(result, "╭"+strings.Repeat("─", innerWidth)+"╮")
	for i := 0; i < height; i++ {
		line := ""
		if i < len(lines) {
			line = lines[i]
		}
		result = append(result, "│"+padOrTruncate(line, innerWidth)+"│")
	}
	result = append(result, "╰"+strings.Repeat("─", innerWidth)+"╯")

	return lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(strings.Join(result, "\n"))
}
