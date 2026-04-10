package render

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	glamouransi "github.com/charmbracelet/glamour/ansi"
)

type Options struct {
	Style string
}

type Heading struct {
	Level      int
	Text       string
	SourceLine int
}

type Document struct {
	Path       string
	Title      string
	Content    string
	Width      int
	Size       int64
	ModTime    time.Time
	Fallback   bool
	LineCount  int
	SourceText string
	Headings   []Heading
}

type Renderer struct {
	style  string
	styles *glamouransi.StyleConfig
	cache  *Cache
}

func NewRenderer(options Options) *Renderer {
	style := options.Style
	var styleConfig *glamouransi.StyleConfig
	if style == "" || style == "dark" {
		config := readerStyleConfig()
		styleConfig = &config
		style = ""
	}
	return &Renderer{
		style:  style,
		styles: styleConfig,
		cache:  NewCache(),
	}
}

func (r *Renderer) RenderTextFile(path string, width int, lang string) (Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Document{}, err
	}

	width = max(width, 20)
	if cached, ok := r.cache.Get(path, width, info.ModTime()); ok {
		return cached, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}

	source := normalizeLineEndings(string(data))
	sourceLines := splitSourceLines(source)

	wrapped := "```" + lang + "\n" + source + "\n```"
	content, fallback := r.render(wrapped, width)

	doc := Document{
		Path:       path,
		Title:      filepath.Base(path),
		Content:    content,
		Width:      width,
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		Fallback:   fallback,
		LineCount:  len(sourceLines),
		SourceText: source,
	}

	r.cache.Set(doc)
	return doc, nil
}

func (r *Renderer) RenderFile(path string, width int) (Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Document{}, err
	}

	width = max(width, 20)
	if cached, ok := r.cache.Get(path, width, info.ModTime()); ok {
		return cached, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}

	source := normalizeLineEndings(string(data))
	sourceLines := splitSourceLines(source)
	content, fallback := r.render(source, width)
	doc := Document{
		Path:       path,
		Title:      filepath.Base(path),
		Content:    content,
		Width:      width,
		Size:       info.Size(),
		ModTime:    info.ModTime(),
		Fallback:   fallback,
		LineCount:  len(sourceLines),
		SourceText: source,
		Headings:   parseHeadings(source),
	}

	r.cache.Set(doc)
	return doc, nil
}

func (r *Renderer) render(source string, width int) (string, bool) {
	source = prepareRenderSource(source)
	options := []glamour.TermRendererOption{
		glamour.WithWordWrap(width),
		glamour.WithPreservedNewLines(),
	}
	if r.styles != nil {
		options = append(options, glamour.WithStyles(*r.styles))
	} else {
		options = append(options, glamour.WithStandardStyle(r.style))
	}

	renderer, err := glamour.NewTermRenderer(options...)
	if err != nil {
		return source, true
	}

	rendered, err := renderer.Render(source)
	if err != nil {
		return source, true
	}

	return normalizeLineEndings(strings.TrimRight(rendered, "\n")), false
}

func normalizeLineEndings(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func splitSourceLines(source string) []string {
	lines := strings.Split(normalizeLineEndings(source), "\n")
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func parseHeadings(source string) []Heading {
	lines := splitSourceLines(source)
	headings := make([]Heading, 0, 8)

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}

		if level, text, ok := parseATXHeading(trimmed); ok {
			headings = append(headings, Heading{Level: level, Text: text, SourceLine: i})
			continue
		}

		if i+1 >= len(lines) {
			continue
		}

		if level, ok := parseSetextUnderline(strings.TrimSpace(lines[i+1])); ok {
			headings = append(headings, Heading{
				Level:      level,
				Text:       trimmed,
				SourceLine: i,
			})
			i++
		}
	}

	return headings
}

func parseATXHeading(line string) (level int, text string, ok bool) {
	count := 0
	for count < len(line) && count < 6 && line[count] == '#' {
		count++
	}
	if count == 0 {
		return 0, "", false
	}
	if count < len(line) && line[count] != ' ' && line[count] != '\t' {
		return 0, "", false
	}

	text = strings.TrimSpace(line[count:])
	text = strings.TrimRight(text, "#")
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, "", false
	}

	return count, text, true
}

func parseSetextUnderline(line string) (level int, ok bool) {
	if line == "" {
		return 0, false
	}

	if strings.Trim(line, "=") == "" {
		return 1, true
	}
	if strings.Trim(line, "-") == "" {
		return 2, true
	}
	return 0, false
}

func prepareRenderSource(source string) string {
	lines := strings.Split(normalizeLineEndings(source), "\n")
	if len(lines) == 0 {
		return source
	}

	inFence := false
	var fenceChar byte
	fenceLen := 0

	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		indentLen := len(line) - len(trimmed)
		indent := line[:indentLen]

		if !inFence {
			char, count, rest, ok := parseFenceDelimiter(trimmed)
			if !ok {
				continue
			}

			if strings.TrimSpace(rest) == "" {
				lines[i] = indent + strings.Repeat(string(char), count) + "plaintext"
			}
			inFence = true
			fenceChar = char
			fenceLen = count
			continue
		}

		if isFenceClosing(trimmed, fenceChar, fenceLen) {
			inFence = false
			fenceChar = 0
			fenceLen = 0
		}
	}

	return strings.Join(lines, "\n")
}

func parseFenceDelimiter(line string) (char byte, count int, rest string, ok bool) {
	if len(line) < 3 {
		return 0, 0, "", false
	}

	char = line[0]
	if char != '`' && char != '~' {
		return 0, 0, "", false
	}

	for count < len(line) && line[count] == char {
		count++
	}
	if count < 3 {
		return 0, 0, "", false
	}

	return char, count, line[count:], true
}

func isFenceClosing(line string, char byte, minCount int) bool {
	fenceChar, count, rest, ok := parseFenceDelimiter(line)
	if !ok || fenceChar != char || count < minCount {
		return false
	}
	return strings.TrimSpace(rest) == ""
}
