package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/lunemis/skimd/internal/browser"
	"github.com/lunemis/skimd/internal/render"
)

func TestModelOpensMarkdownAndScrolls(t *testing.T) {
	root := t.TempDir()
	content := "# Title\n\n"
	for i := 0; i < 80; i++ {
		content += "line " + strings.Repeat("x", 20) + "\n"
	}
	path := filepath.Join(root, "doc.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})

	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = model.(Model)

	dirMsg := loadDirectoryCmd(root, "", false)().(dirLoadedMsg)
	model, cmd := m.Update(dirMsg)
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected initial hover preview render command")
	}
	previewMsg := cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	if len(m.entries) == 0 {
		t.Fatalf("expected directory entries")
	}
	if m.preview.Path != path {
		t.Fatalf("expected hover preview path %q, got %q", path, m.preview.Path)
	}
	if m.focus != focusBrowser {
		t.Fatalf("expected focusBrowser during hover preview, got %v", m.focus)
	}
	entry, ok := m.currentEntry()
	if !ok || entry.Path != path {
		t.Fatalf("expected current entry to focus preview file %q, got %+v", path, entry)
	}

	normalBrowserWidth := m.browserWidth()

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected reader mode render command")
	}

	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	if m.preview.Path != path {
		t.Fatalf("expected preview path %q, got %q", path, m.preview.Path)
	}
	if m.focus != focusPreview {
		t.Fatalf("expected focusPreview, got %v", m.focus)
	}
	if !m.readerMode {
		t.Fatalf("expected readerMode to be enabled after enter")
	}
	if m.browserWidth() >= normalBrowserWidth {
		t.Fatalf("expected browser width to shrink in reader mode, before=%d after=%d", normalBrowserWidth, m.browserWidth())
	}
	compactBrowserWidth := m.browserWidth()

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = model.(Model)
	if !m.zenMode {
		t.Fatalf("expected zenMode to be enabled after z")
	}
	if m.browserWidth() != 0 {
		t.Fatalf("expected browser width 0 in zen mode, got %d", m.browserWidth())
	}
	if cmd == nil {
		t.Fatalf("expected rerender command when entering zen mode")
	}

	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = model.(Model)
	if m.zenMode {
		t.Fatalf("expected zenMode to be disabled after second z")
	}
	if m.browserWidth() != compactBrowserWidth {
		t.Fatalf("expected browser width to restore after zen mode, before=%d after=%d", compactBrowserWidth, m.browserWidth())
	}
	if cmd == nil {
		t.Fatalf("expected rerender command when leaving zen mode")
	}

	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	before := m.previewOffset
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = model.(Model)
	if m.previewOffset <= before {
		t.Fatalf("expected preview offset to increase, before=%d after=%d", before, m.previewOffset)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(Model)
	if m.readerMode {
		t.Fatalf("expected readerMode to be disabled after esc")
	}
	if m.zenMode {
		t.Fatalf("expected zenMode to be disabled after esc")
	}
	if m.focus != focusBrowser {
		t.Fatalf("expected focusBrowser after esc, got %v", m.focus)
	}
	if m.browserWidth() != normalBrowserWidth {
		t.Fatalf("expected browser width to restore after esc, before=%d after=%d", normalBrowserWidth, m.browserWidth())
	}
	if cmd == nil {
		t.Fatalf("expected preview rerender command after leaving reader mode")
	}
}

func TestViewMainFitsHeight(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 120
	m.height = 30
	m.entries = []browser.Entry{
		{Name: "..", Path: "/", Kind: browser.EntryParent},
		{Name: "docs", Path: "/tmp/docs", Kind: browser.EntryDirectory},
		{Name: "readme.md", Path: "/tmp/readme.md", Kind: browser.EntryMarkdown, Size: 512},
	}
	m.status = "Ready"
	m.preview.Path = "/tmp/readme.md"
	m.previewLines = make([]string, 50)
	for i := range m.previewLines {
		m.previewLines[i] = "preview line"
	}

	output := m.viewMain()
	lines := strings.Split(output, "\n")
	if len(lines) > m.height {
		t.Fatalf("expected view to fit height %d, got %d lines", m.height, len(lines))
	}
}

func TestViewMainFitsHeightInZenMode(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 120
	m.height = 30
	m.readerMode = true
	m.zenMode = true
	m.preview.Path = "/tmp/readme.md"
	m.previewLines = make([]string, 80)
	for i := range m.previewLines {
		m.previewLines[i] = "preview line"
	}

	output := m.viewMain()
	lines := strings.Split(output, "\n")
	if len(lines) > m.height {
		t.Fatalf("expected zen view to fit height %d, got %d lines", m.height, len(lines))
	}
}

func TestPreviewRenderWidthCapsReaderContent(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 220
	m.height = 30

	width := m.previewRenderWidth()
	if width != previewMaxContentWidth {
		t.Fatalf("expected hover preview width cap %d, got %d", previewMaxContentWidth, width)
	}

	m.readerMode = true
	width = m.previewRenderWidth()
	if width != readerAutoMaxContentWidth {
		t.Fatalf("expected adaptive reader preview width %d, got %d", readerAutoMaxContentWidth, width)
	}
}

func TestPreviewMetaLineIncludesCurrentSection(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.preview = render.Document{
		Path:      "/tmp/readme.md",
		Size:      2048,
		LineCount: 120,
		ModTime:   time.Now().Add(-10 * time.Minute),
		Headings: []render.Heading{
			{Level: 1, Text: "Intro", SourceLine: 0},
			{Level: 2, Text: "Usage", SourceLine: 30},
		},
	}
	m.outline = []outlineEntry{
		{Level: 1, Text: "Intro", PreviewLine: 0, SourceLine: 0},
		{Level: 2, Text: "Usage", PreviewLine: 30, SourceLine: 30},
	}
	m.previewLines = make([]string, 120)
	m.previewOffset = 35

	meta := m.previewMetaLine()
	if !strings.Contains(meta, "§ Usage") {
		t.Fatalf("expected current section in meta line, got %q", meta)
	}
}

func TestBrowserModeLabelUsesDocsAndFiles(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	if got := m.browserModeLabel(); got != "Docs" {
		t.Fatalf("expected default browser label Docs, got %q", got)
	}

	m.showAllFiles = true
	if got := m.browserModeLabel(); got != "Files" {
		t.Fatalf("expected all-files browser label Files, got %q", got)
	}
}

func TestRenderHelpUsesBrowserContext(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 160

	help := m.renderHelp()
	if !strings.Contains(help, "filter") {
		t.Fatalf("expected browser help to include filter action, got %q", help)
	}
	if !strings.Contains(help, "Enter") || !strings.Contains(help, "open") {
		t.Fatalf("expected browser help to include open action, got %q", help)
	}
	if !strings.Contains(help, "all files") {
		t.Fatalf("expected browser help to include all files toggle, got %q", help)
	}
	if strings.Contains(help, "search") {
		t.Fatalf("expected browser help to hide reader search, got %q", help)
	}

	m.showAllFiles = true
	help = m.renderHelp()
	if !strings.Contains(help, "docs only") {
		t.Fatalf("expected browser help to describe return toggle, got %q", help)
	}
}

func TestRenderHelpUsesBrowserFilterContext(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 160
	m.browserFilterInputActive = true

	help := m.renderHelp()
	if !strings.Contains(help, "query") || !strings.Contains(help, "apply") || !strings.Contains(help, "close") {
		t.Fatalf("expected browser filter help to show filter editing actions, got %q", help)
	}
	if strings.Contains(help, "search") {
		t.Fatalf("expected browser filter help to hide reader search actions, got %q", help)
	}
}

func TestRenderHelpUsesReaderContext(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 160
	m.readerMode = true

	help := m.renderHelp()
	if !strings.Contains(help, "search") || !strings.Contains(help, "prev") || !strings.Contains(help, "next") {
		t.Fatalf("expected reader help to include search and section jump, got %q", help)
	}
	if !strings.Contains(help, "width") {
		t.Fatalf("expected reader help to include width controls, got %q", help)
	}
	if strings.Contains(help, "all files") || strings.Contains(help, "docs only") {
		t.Fatalf("expected reader help to hide browser-only file toggle, got %q", help)
	}
}

func TestRenderHelpUsesCompactReaderContextOnNarrowTerminals(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 80
	m.readerMode = true

	help := m.renderHelp()
	if !strings.Contains(help, "search") || !strings.Contains(help, "prev") || !strings.Contains(help, "next") {
		t.Fatalf("expected narrow reader help to keep core reader actions, got %q", help)
	}
	if strings.Contains(help, "width") || strings.Contains(help, "outline") {
		t.Fatalf("expected narrow reader help to drop secondary actions, got %q", help)
	}
}

func TestAdjustReaderWidthRerendersPreview(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "doc.md")
	if err := os.WriteFile(path, []byte("# Title\n\nbody\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})
	m.width = 220
	m.height = 30
	m.readerMode = true
	m.focus = focusPreview
	m.preview = render.Document{Path: path, Width: m.previewRenderWidth()}

	before := m.previewRenderWidth()
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'='}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected rerender command after widening reader")
	}
	if got := m.previewRenderWidth(); got <= before {
		t.Fatalf("expected reader width to increase, before=%d after=%d", before, got)
	}
	if !strings.Contains(m.status, "Reading width") {
		t.Fatalf("expected width status message, got %q", m.status)
	}

	before = m.previewRenderWidth()
	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected rerender command after narrowing reader")
	}
	if got := m.previewRenderWidth(); got >= before {
		t.Fatalf("expected reader width to decrease, before=%d after=%d", before, got)
	}
}

func TestRenderPreviewBodyMarksOnlyCurrentSearchResultLine(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.readerMode = true
	m.searchQuery = "alpha"
	m.searchMatches = []int{0, 1}
	m.searchMatchIndex = 1
	m.previewLines = []string{
		"\x1b[31malpha\x1b[0m beta",
		"beta alpha",
		"gamma",
	}
	m.plainPreview = []string{
		"alpha beta",
		"beta alpha",
		"gamma",
	}

	body := m.renderPreviewBody(40, 0, 3)
	lines := strings.Split(body, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 rendered lines, got %d", len(lines))
	}
	if strings.Contains(lines[0], "▎") {
		t.Fatalf("expected non-current search line to stay unmarked, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "▎") {
		t.Fatalf("expected current search line to show marker, got %q", lines[1])
	}
	if stripped := ansi.Strip(lines[1]); !strings.Contains(stripped, "beta alpha") {
		t.Fatalf("expected current search line text to remain intact, got %q", stripped)
	}
	if !strings.Contains(lines[1], "\x1b[") {
		t.Fatalf("expected current search line to contain highlight styling, got %q", lines[1])
	}
	if strings.Contains(lines[2], "▎") {
		t.Fatalf("expected non-matching line to stay unmarked, got %q", lines[2])
	}
}

func TestRenderHelpUsesSearchContext(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 160
	m.readerMode = true
	m.searchInputActive = true

	help := m.renderHelp()
	if !strings.Contains(help, "type") || !strings.Contains(help, "apply") || !strings.Contains(help, "cancel") {
		t.Fatalf("expected search help to show query editing actions, got %q", help)
	}
	if strings.Contains(help, "outline view") {
		t.Fatalf("expected search help to replace normal reader help, got %q", help)
	}
}

func TestRenderOutlineContentMarksCurrentAndSelectedRows(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.outline = []outlineEntry{
		{Level: 1, Text: "Intro", PreviewLine: 0, SourceLine: 0},
		{Level: 2, Text: "Usage", PreviewLine: 20, SourceLine: 20},
		{Level: 2, Text: "Notes", PreviewLine: 40, SourceLine: 40},
	}
	m.preview = render.Document{LineCount: 60, Headings: []render.Heading{
		{Level: 1, Text: "Intro", SourceLine: 0},
		{Level: 2, Text: "Usage", SourceLine: 20},
		{Level: 2, Text: "Notes", SourceLine: 40},
	}}
	m.previewLines = make([]string, 60)
	m.outlineView = outlineFull
	m.previewOffset = 25
	m.outlineCursor = 2

	content := m.renderOutlineContent(24, 3)
	lines := strings.Split(content, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 outline lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "• ") {
		t.Fatalf("expected current outline row to include current marker, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "› ") {
		t.Fatalf("expected selected outline row to include selection marker, got %q", lines[2])
	}
}

func TestRenderOutlineContentKeepsCurrentMarkerWhileReading(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.outline = []outlineEntry{
		{Level: 1, Text: "Intro", PreviewLine: 0, SourceLine: 0},
		{Level: 2, Text: "Usage", PreviewLine: 20, SourceLine: 20},
		{Level: 2, Text: "Notes", PreviewLine: 40, SourceLine: 40},
	}
	m.preview = render.Document{LineCount: 60, Headings: []render.Heading{
		{Level: 1, Text: "Intro", SourceLine: 0},
		{Level: 2, Text: "Usage", SourceLine: 20},
		{Level: 2, Text: "Notes", SourceLine: 40},
	}}
	m.previewLines = make([]string, 60)
	m.outlineView = outlineSide
	m.previewOffset = 25
	m.outlineCursor = 2

	content := m.renderOutlineContent(24, 3)
	lines := strings.Split(content, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 outline lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "• ") {
		t.Fatalf("expected current outline row to include current marker, got %q", lines[1])
	}
	if strings.Contains(lines[2], "› ") {
		t.Fatalf("expected reading mode to hide selection marker, got %q", lines[2])
	}
}

func TestRenderOutlineContentShowsSelectionWhenSideOutlineFocused(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.outline = []outlineEntry{
		{Level: 1, Text: "Intro", PreviewLine: 0, SourceLine: 0},
		{Level: 2, Text: "Usage", PreviewLine: 20, SourceLine: 20},
		{Level: 2, Text: "Notes", PreviewLine: 40, SourceLine: 40},
	}
	m.preview = render.Document{LineCount: 60, Headings: []render.Heading{
		{Level: 1, Text: "Intro", SourceLine: 0},
		{Level: 2, Text: "Usage", SourceLine: 20},
		{Level: 2, Text: "Notes", SourceLine: 40},
	}}
	m.previewLines = make([]string, 60)
	m.readerMode = true
	m.focus = focusOutline
	m.outlineView = outlineSide
	m.previewOffset = 25
	m.outlineCursor = 2

	content := m.renderOutlineContent(24, 3)
	lines := strings.Split(content, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 outline lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "• ") {
		t.Fatalf("expected current outline row to include current marker, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "› ") {
		t.Fatalf("expected focused side outline to show selection marker, got %q", lines[2])
	}
}

func TestRenderHelpUsesSideOutlineContext(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.width = 160
	m.readerMode = true
	m.focus = focusOutline
	m.outlineView = outlineSide

	help := m.renderHelp()
	if !strings.Contains(help, "jump") || !strings.Contains(help, "read") {
		t.Fatalf("expected side outline help to include jump/read actions, got %q", help)
	}
	if strings.Contains(help, "search") {
		t.Fatalf("expected side outline help to hide reader search actions, got %q", help)
	}
}

func TestSideOutlineArrowNavigation(t *testing.T) {
	m := NewModel(browser.StartLocation{Dir: "/tmp"})
	m.readerMode = true
	m.focus = focusPreview
	m.outlineView = outlineSide
	m.preview = render.Document{
		Path:      "/tmp/readme.md",
		LineCount: 60,
		Headings: []render.Heading{
			{Level: 1, Text: "Intro", SourceLine: 0},
			{Level: 2, Text: "Usage", SourceLine: 20},
			{Level: 2, Text: "Notes", SourceLine: 40},
		},
	}
	m.previewLines = make([]string, 60)
	m.outline = []outlineEntry{
		{Level: 1, Text: "Intro", PreviewLine: 0, SourceLine: 0},
		{Level: 2, Text: "Usage", PreviewLine: 20, SourceLine: 20},
		{Level: 2, Text: "Notes", PreviewLine: 40, SourceLine: 40},
	}
	m.previewOffset = 0

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = model.(Model)
	if m.focus != focusOutline {
		t.Fatalf("expected left arrow to move focus to side outline, got %v", m.focus)
	}
	if m.outlineCursor != 0 {
		t.Fatalf("expected side outline focus to start at current heading, got %d", m.outlineCursor)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = model.(Model)
	if m.outlineCursor != 1 {
		t.Fatalf("expected side outline cursor to move down, got %d", m.outlineCursor)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if m.focus != focusPreview {
		t.Fatalf("expected enter on side outline to return focus to reader, got %v", m.focus)
	}
	if m.outlineView != outlineSide {
		t.Fatalf("expected side outline to remain open after heading jump")
	}
	if m.previewOffset != 20 {
		t.Fatalf("expected enter on side outline to jump reader to heading, got %d", m.previewOffset)
	}
}

func TestModelSearchAndOutline(t *testing.T) {
	root := t.TempDir()
	lines := []string{
		"# Intro",
		"",
		"needleunique alpha",
	}
	for i := 0; i < 20; i++ {
		lines = append(lines, "intro filler line")
	}
	lines = append(lines,
		"",
		"## Usage",
		"",
		"body",
		"needleunique beta",
	)
	for i := 0; i < 20; i++ {
		lines = append(lines, "usage filler line")
	}
	lines = append(lines,
		"",
		"## Notes",
		"",
		"final section",
	)
	content := strings.Join(lines, "\n")
	path := filepath.Join(root, "doc.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})
	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 14})
	m = model.(Model)

	dirMsg := loadDirectoryCmd(root, "", false)().(dirLoadedMsg)
	model, cmd := m.Update(dirMsg)
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected initial preview render command")
	}

	previewMsg := cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected reader render command")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	if len(m.outline) < 3 {
		t.Fatalf("expected outline entries, got %d", len(m.outline))
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = model.(Model)
	if m.outlineView != outlineFull {
		t.Fatalf("expected full outline mode, got %v", m.outlineView)
	}
	if m.browserWidth() != 0 {
		t.Fatalf("expected browser width 0 in full outline mode, got %d", m.browserWidth())
	}
	if cmd != nil {
		previewMsg = cmd().(previewLoadedMsg)
		model, _ = m.Update(previewMsg)
		m = model.(Model)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = model.(Model)
	if m.outlineView != outlineSide {
		t.Fatalf("expected side outline mode, got %v", m.outlineView)
	}
	if m.focus != focusPreview {
		t.Fatalf("expected preview focus in side outline mode, got %v", m.focus)
	}
	if m.browserWidth() == 0 {
		t.Fatalf("expected side outline to keep a left pane")
	}
	if !strings.Contains(m.viewMain(), "[ Outline ]") {
		t.Fatalf("expected outline panel to be visible in side outline mode")
	}
	if cmd != nil {
		previewMsg = cmd().(previewLoadedMsg)
		model, _ = m.Update(previewMsg)
		m = model.(Model)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = model.(Model)
	if m.outlineCursor != m.currentOutlineIndex() {
		t.Fatalf("expected side outline to follow preview, cursor=%d current=%d", m.outlineCursor, m.currentOutlineIndex())
	}

	previousLine := m.previewOffset
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})
	m = model.(Model)
	if m.outlineView != outlineSide {
		t.Fatalf("expected side outline to remain open after heading jump")
	}
	if m.previewOffset >= previousLine {
		t.Fatalf("expected previous heading jump to move upward, before=%d after=%d", previousLine, m.previewOffset)
	}
	if m.outlineCursor != m.currentOutlineIndex() {
		t.Fatalf("expected outline cursor to stay synced after heading jump, cursor=%d current=%d", m.outlineCursor, m.currentOutlineIndex())
	}

	previousLine = m.previewOffset
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	m = model.(Model)
	if m.previewOffset <= previousLine {
		t.Fatalf("expected next heading jump to move downward, before=%d after=%d", previousLine, m.previewOffset)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = model.(Model)
	if !m.searchInputActive {
		t.Fatalf("expected search input mode")
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("needleunique")})
	m = model.(Model)
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if len(m.searchMatches) < 2 {
		t.Fatalf("expected multiple search matches, got %d", len(m.searchMatches))
	}

	firstOffset := m.previewOffset
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = model.(Model)
	if m.previewOffset == firstOffset {
		t.Fatalf("expected n to move to next search match")
	}
}

func TestModelReviewModePrefersDirectoryWhenPresentAndCanShowAllFiles(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	oldPath := filepath.Join(root, "old.md")
	newPath := filepath.Join(root, "new.md")
	otherPath := filepath.Join(root, "notes.txt")

	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}
	if err := os.WriteFile(oldPath, []byte("# old"), 0o644); err != nil {
		t.Fatalf("WriteFile(old) error = %v", err)
	}
	if err := os.WriteFile(newPath, []byte("# new"), 0o644); err != nil {
		t.Fatalf("WriteFile(new) error = %v", err)
	}
	if err := os.WriteFile(otherPath, []byte("plain"), 0o644); err != nil {
		t.Fatalf("WriteFile(other) error = %v", err)
	}

	oldTime := time.Now().Add(-2 * time.Hour)
	newTime := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(old) error = %v", err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatalf("Chtimes(new) error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})
	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m = model.(Model)

	dirMsg := loadDirectoryCmd(root, "", false)().(dirLoadedMsg)
	model, cmd := m.Update(dirMsg)
	m = model.(Model)
	if cmd != nil {
		t.Fatalf("expected no initial preview command when a directory is present")
	}

	entry, ok := m.currentEntry()
	if !ok {
		t.Fatalf("expected current entry")
	}
	if entry.Name != "docs" || !entry.IsDirectory() {
		t.Fatalf("expected directory to be focused first, got %+v", entry)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected preview command after moving from directory to markdown")
	}

	previewMsg := cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	entry, ok = m.currentEntry()
	if entry.Name != "new.md" {
		t.Fatalf("expected newest markdown after moving down, got %q", entry.Name)
	}
	if m.preview.Path != newPath {
		t.Fatalf("expected preview path %q, got %q", newPath, m.preview.Path)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = model.(Model)
	if !m.showAllFiles {
		t.Fatalf("expected showAllFiles after pressing a")
	}
	if cmd == nil {
		t.Fatalf("expected reload command after toggling all files")
	}

	dirMsg = cmd().(dirLoadedMsg)
	model, _ = m.Update(dirMsg)
	m = model.(Model)

	foundText := false
	for _, item := range m.entries {
		if item.Name == "notes.txt" && item.Kind == browser.EntryText {
			foundText = true
			break
		}
	}
	if !foundText {
		t.Fatalf("expected notes.txt to appear in all-files mode")
	}
}

func TestModelBrowserFilterFiltersEntriesAndClears(t *testing.T) {
	root := t.TempDir()
	authPath := filepath.Join(root, "AUTH_SERVICE.md")
	otherPath := filepath.Join(root, "README.md")
	dirPath := filepath.Join(root, "auth-notes")

	if err := os.WriteFile(authPath, []byte("# auth"), 0o644); err != nil {
		t.Fatalf("WriteFile(auth) error = %v", err)
	}
	if err := os.WriteFile(otherPath, []byte("# readme"), 0o644); err != nil {
		t.Fatalf("WriteFile(readme) error = %v", err)
	}
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(auth-notes) error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})
	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	m = model.(Model)

	dirMsg := loadDirectoryCmd(root, "", false)().(dirLoadedMsg)
	model, cmd := m.Update(dirMsg)
	m = model.(Model)
	if cmd != nil {
		t.Fatalf("expected no initial hover preview when a directory is focused first")
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = model.(Model)
	if !m.browserFilterInputActive {
		t.Fatalf("expected browser filter input mode")
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("auth")})
	m = model.(Model)
	if cmd != nil {
		t.Fatalf("expected no hover preview while filtered selection is still a directory")
	}
	if m.browserFilterQuery != "auth" {
		t.Fatalf("expected browser filter query auth, got %q", m.browserFilterQuery)
	}
	if len(m.entries) != 3 {
		t.Fatalf("expected parent plus 2 auth matches, got %d entries", len(m.entries))
	}
	if m.entries[1].Name != "auth-notes" || m.entries[2].Name != "AUTH_SERVICE.md" {
		t.Fatalf("expected filtered entries to keep matching dir/file, got %+v", m.entries)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if m.browserFilterInputActive {
		t.Fatalf("expected enter to leave browser filter input mode")
	}
	if !strings.Contains(m.renderInfoLine(), "2 matches") {
		t.Fatalf("expected filter summary after apply, got %q", m.renderInfoLine())
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview command after moving from filtered directory to markdown")
	}
	previewMsg := cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)
	if m.preview.Path != authPath {
		t.Fatalf("expected filtered hover preview path %q, got %q", authPath, m.preview.Path)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(Model)
	if cmd != nil {
		previewMsg = cmd().(previewLoadedMsg)
		model, _ = m.Update(previewMsg)
		m = model.(Model)
	}
	if m.browserFilterQuery != "" {
		t.Fatalf("expected browser filter query to clear, got %q", m.browserFilterQuery)
	}
	if len(m.entries) < 4 {
		t.Fatalf("expected full entry list after clearing filter, got %d entries", len(m.entries))
	}
	if m.preview.Path != authPath {
		t.Fatalf("expected current preview to stay on %q after clearing filter, got %q", authPath, m.preview.Path)
	}
}

func TestModelRestoresPreviewOffsetWithinSameDirectory(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "alpha.md")
	secondPath := filepath.Join(root, "beta.md")

	if err := os.WriteFile(firstPath, []byte(longMarkdown("Alpha")), 0o644); err != nil {
		t.Fatalf("WriteFile(alpha) error = %v", err)
	}
	if err := os.WriteFile(secondPath, []byte(longMarkdown("Beta")), 0o644); err != nil {
		t.Fatalf("WriteFile(beta) error = %v", err)
	}

	newer := time.Now().Add(-2 * time.Minute)
	older := time.Now().Add(-10 * time.Minute)
	if err := os.Chtimes(firstPath, newer, newer); err != nil {
		t.Fatalf("Chtimes(alpha) error = %v", err)
	}
	if err := os.Chtimes(secondPath, older, older); err != nil {
		t.Fatalf("Chtimes(beta) error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})
	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 14})
	m = model.(Model)

	dirMsg := loadDirectoryCmd(root, "alpha.md", false)().(dirLoadedMsg)
	model, cmd := m.Update(dirMsg)
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview command for alpha")
	}

	previewMsg := cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected reader render command for alpha")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = model.(Model)
	savedSourceLine := m.currentSourceLine()
	if savedSourceLine == 0 {
		t.Fatalf("expected alpha preview offset to move away from top")
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(Model)
	if cmd != nil {
		previewMsg = cmd().(previewLoadedMsg)
		model, _ = m.Update(previewMsg)
		m = model.(Model)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview command for beta")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)
	if m.preview.Path != secondPath {
		t.Fatalf("expected beta preview, got %q", m.preview.Path)
	}
	if got := m.currentSourceLine(); got != 0 {
		t.Fatalf("expected beta preview to start from source line 0, got %d", got)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview command when returning to alpha")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)
	if m.preview.Path != firstPath {
		t.Fatalf("expected alpha preview, got %q", m.preview.Path)
	}
	if got := m.currentSourceLine(); got != savedSourceLine {
		t.Fatalf("expected alpha preview source line %d, got %d", savedSourceLine, got)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected reader render command when reopening alpha")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)
	if got := m.currentSourceLine(); got != savedSourceLine {
		t.Fatalf("expected alpha reader source line %d after reopen, got %d", savedSourceLine, got)
	}
}

func TestModelResetsPreviewOffsetsAfterDirectoryChange(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "docs")
	path := filepath.Join(root, "alpha.md")

	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll(docs) error = %v", err)
	}
	if err := os.WriteFile(path, []byte(longMarkdown("Alpha")), 0o644); err != nil {
		t.Fatalf("WriteFile(alpha) error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})
	model, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 14})
	m = model.(Model)

	dirMsg := loadDirectoryCmd(root, "alpha.md", false)().(dirLoadedMsg)
	model, cmd := m.Update(dirMsg)
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview command for alpha")
	}

	previewMsg := cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected reader render command for alpha")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = model.(Model)
	if m.previewOffset == 0 {
		t.Fatalf("expected alpha preview offset to move away from top")
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(Model)
	if cmd != nil {
		previewMsg = cmd().(previewLoadedMsg)
		model, _ = m.Update(previewMsg)
		m = model.(Model)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = model.(Model)
	entry, ok := m.currentEntry()
	if !ok || !entry.IsDirectory() {
		t.Fatalf("expected directory entry to be selected, got %+v", entry)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected enter-directory command")
	}
	dirMsg = cmd().(dirLoadedMsg)
	model, _ = m.Update(dirMsg)
	m = model.(Model)

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected return-to-parent command")
	}
	dirMsg = cmd().(dirLoadedMsg)
	model, _ = m.Update(dirMsg)
	m = model.(Model)

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview command after returning to root")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	if m.preview.Path != path {
		t.Fatalf("expected alpha preview after returning to root, got %q", m.preview.Path)
	}
	if got := m.currentSourceLine(); got != 0 {
		t.Fatalf("expected alpha preview anchor to reset after directory change, got source line %d", got)
	}
}

func TestReaderToPreviewKeepsSectionAndSourceProgressAcrossWidths(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "doc.md")
	content := strings.Join([]string{
		"# Root",
		"",
		"intro " + strings.Repeat("alpha beta gamma ", 8),
		"",
		"## 2. Reader",
		"",
		"reader body " + strings.Repeat("common phrase for wrapping ", 10),
		"reader body " + strings.Repeat("common phrase for wrapping ", 10),
		"",
		"### 2.1 Stable Section",
		"",
		"| Method | Path | Description |",
		"| --- | --- | --- |",
		"| GET | /api/* | " + strings.Repeat("same cell content ", 6) + " |",
		"| POST | /signup-requests | " + strings.Repeat("same cell content ", 6) + " |",
		"",
		"stable paragraph " + strings.Repeat("shared terms for ambiguous matching ", 10),
		"stable paragraph " + strings.Repeat("shared terms for ambiguous matching ", 10),
		"",
		"## 7. Later",
		"",
		"later body " + strings.Repeat("common phrase for wrapping ", 10),
		"",
		"### 7.1 Different Section",
		"",
		"later paragraph " + strings.Repeat("shared terms for ambiguous matching ", 10),
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m := NewModel(browser.StartLocation{Dir: root})
	model, _ := m.Update(tea.WindowSizeMsg{Width: 170, Height: 26})
	m = model.(Model)

	dirMsg := loadDirectoryCmd(root, "doc.md", false)().(dirLoadedMsg)
	model, cmd := m.Update(dirMsg)
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview command")
	}
	previewMsg := cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected reader preview command")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	target := -1
	for i, entry := range m.outline {
		if entry.Text == "2.1 Stable Section" {
			target = i
			break
		}
	}
	if target < 0 {
		t.Fatalf("expected outline entry for target section")
	}

	m.jumpToPreviewLine(m.outline[target].PreviewLine + 4)
	savedSection := m.currentSectionLabel()
	savedSourceLine := m.currentSourceLine()
	if savedSection != "2.1 Stable Section" {
		t.Fatalf("expected reader section to be 2.1 Stable Section, got %q", savedSection)
	}

	model, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = model.(Model)
	if cmd == nil {
		t.Fatalf("expected hover preview rerender after leaving reader")
	}
	previewMsg = cmd().(previewLoadedMsg)
	model, _ = m.Update(previewMsg)
	m = model.(Model)

	if got := m.currentSectionLabel(); got != savedSection {
		t.Fatalf("expected preview section %q after width change, got %q", savedSection, got)
	}
	if diff := absInt(m.currentSourceLine() - savedSourceLine); diff > 2 {
		t.Fatalf("expected preview source line to stay near %d, got %d (diff=%d)", savedSourceLine, m.currentSourceLine(), diff)
	}
	if m.preview.LineCount != 26 {
		t.Fatalf("expected source line count 26, got %d", m.preview.LineCount)
	}
}

func longMarkdown(title string) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	for i := 0; i < 48; i++ {
		builder.WriteString("line ")
		builder.WriteString(strings.Repeat("x", 24))
		builder.WriteByte('\n')
	}
	return builder.String()
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
