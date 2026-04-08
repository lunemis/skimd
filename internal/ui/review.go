package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/lunemis/skimd/internal/render"
)

type outlineEntry struct {
	Level       int
	Text        string
	PreviewLine int
	SourceLine  int
}

func buildOutline(doc render.Document, previewLines []string) []outlineEntry {
	if len(doc.Headings) == 0 {
		return nil
	}

	plain := plainLines(previewLines)
	outline := make([]outlineEntry, 0, len(doc.Headings))
	searchStart := 0

	for _, heading := range doc.Headings {
		line := findLineContaining(plain, heading.Text, searchStart)
		if line >= 0 {
			searchStart = line
		} else {
			line = 0
		}

		outline = append(outline, outlineEntry{
			Level:       heading.Level,
			Text:        heading.Text,
			PreviewLine: line,
			SourceLine:  heading.SourceLine,
		})
	}

	return outline
}

func plainLines(lines []string) []string {
	plain := make([]string, len(lines))
	for i, line := range lines {
		plain[i] = ansi.Strip(line)
	}
	return plain
}

func findSearchMatches(lines []string, query string) []int {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	matches := make([]int, 0, 8)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			matches = append(matches, i)
		}
	}
	return matches
}

type searchMatchRange struct {
	Start int
	End   int
}

func findSearchMatchRanges(line, query string) []searchMatchRange {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	lineRunes := []rune(line)
	queryRunes := []rune(query)
	if len(queryRunes) == 0 || len(lineRunes) < len(queryRunes) {
		return nil
	}

	matches := make([]searchMatchRange, 0, 4)
	for i := 0; i+len(queryRunes) <= len(lineRunes); {
		if strings.EqualFold(string(lineRunes[i:i+len(queryRunes)]), query) {
			matches = append(matches, searchMatchRange{Start: i, End: i + len(queryRunes)})
			i += len(queryRunes)
			continue
		}
		i++
	}
	return matches
}

func highlightSearchText(line, query string) string {
	matches := findSearchMatchRanges(line, query)
	if len(matches) == 0 {
		return line
	}

	lineRunes := []rune(line)
	var builder strings.Builder
	cursor := 0

	for _, match := range matches {
		if match.Start > cursor {
			builder.WriteString(string(lineRunes[cursor:match.Start]))
		}
		builder.WriteString(renderSearchMatch(string(lineRunes[match.Start:match.End])))
		cursor = match.End
	}

	if cursor < len(lineRunes) {
		builder.WriteString(string(lineRunes[cursor:]))
	}
	return builder.String()
}

func renderSearchMatch(segment string) string {
	if segment == "" {
		return segment
	}
	return "\x1b[1;7m" + segment + "\x1b[0m"
}

func findLineContaining(lines []string, needle string, start int) int {
	if needle == "" {
		return -1
	}
	needle = strings.ToLower(strings.TrimSpace(needle))

	for i := max(0, start); i < len(lines); i++ {
		if strings.Contains(strings.ToLower(lines[i]), needle) {
			return i
		}
	}
	for i := 0; i < min(start, len(lines)); i++ {
		if strings.Contains(strings.ToLower(lines[i]), needle) {
			return i
		}
	}
	return -1
}
