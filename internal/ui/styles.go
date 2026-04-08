package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorAccent   = lipgloss.Color("#6EE7B7")
	colorBorder   = lipgloss.Color("#4B5563")
	colorSelected = lipgloss.Color("#1F2937")
	colorCursor   = lipgloss.Color("#FDE68A")
	colorMuted    = lipgloss.Color("#94A3B8")
	colorDanger   = lipgloss.Color("#FCA5A5")
	colorTitle    = lipgloss.Color("#E2E8F0")
	colorDir      = lipgloss.Color("#93C5FD")
	colorMarkdown = lipgloss.Color("#FDE68A")
	colorOther    = lipgloss.Color("#CBD5E1")
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorTitle)

	subtleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	searchMarkerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorAccent)

	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorDanger)
)
