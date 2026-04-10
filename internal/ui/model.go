package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/lunemis/skimd/internal/browser"
	"github.com/lunemis/skimd/internal/render"
)

type focusArea int

const (
	focusBrowser focusArea = iota
	focusPreview
	focusOutline
)

type outlineViewMode int

const (
	outlineHidden outlineViewMode = iota
	outlineFull
	outlineSide
)

const (
	previewMaxContentWidth      = 76
	readerBaseContentWidth      = 92
	readerAutoMaxContentWidth   = 128
	readerManualMaxContentWidth = 160
	readerMinContentWidth       = 72
	readerContentWidthStep      = 8
)

type dirLoadedMsg struct {
	path    string
	focus   string
	showAll bool
	items   []browser.Entry
	err     error
}

type previewLoadedMsg struct {
	path  string
	width int
	doc   render.Document
	err   error
}

type previewStatMsg struct {
	path    string
	size    int64
	modTime time.Time
	err     error
}

type tickMsg time.Time

type Model struct {
	width  int
	height int

	currentDir   string
	allEntries   []browser.Entry
	entries      []browser.Entry
	cursor       int
	focus        focusArea
	showAllFiles bool
	readerMode   bool
	zenMode      bool

	renderer *render.Renderer

	preview       render.Document
	previewOffset int
	previewLines  []string
	plainPreview  []string
	outline       []outlineEntry
	outlineView   outlineViewMode
	outlineCursor int

	searchInputActive        bool
	searchQuery              string
	searchMatches            []int
	searchMatchIndex         int
	browserFilterInputActive bool
	browserFilterQuery       string
	readerWidthOffset        int
	previewAnchors           map[string]int

	status string

	pendingOpenPath   string
	pendingFocusName  string
	pendingSourceLine int

	activePreviewPath  string
	activePreviewWidth int
}

func NewModel(start browser.StartLocation) Model {
	return Model{
		currentDir:       start.Dir,
		focus:            focusBrowser,
		showAllFiles:     false,
		readerMode:       start.OpenFile != "",
		renderer:         render.NewRenderer(render.Options{}),
		pendingOpenPath:  start.OpenFile,
		pendingFocusName: start.Focus,
		previewAnchors:   make(map[string]int),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadDirectoryCmd(m.currentDir, m.pendingFocusName, m.showAllFiles),
		tick(),
	)
}

func loadDirectoryCmd(path, focus string, showAll bool) tea.Cmd {
	return func() tea.Msg {
		items, err := browser.ReadDirectory(path, browser.ReadOptions{ShowAllFiles: showAll})
		return dirLoadedMsg{
			path:    path,
			focus:   focus,
			showAll: showAll,
			items:   items,
			err:     err,
		}
	}
}

func renderPreviewCmd(renderer *render.Renderer, path string, width int) tea.Cmd {
	return func() tea.Msg {
		var doc render.Document
		var err error
		if browser.IsTextFile(path) {
			lang := browser.TextFileLang(path)
			doc, err = renderer.RenderTextFile(path, width, lang)
		} else {
			doc, err = renderer.RenderFile(path, width)
		}
		return previewLoadedMsg{
			path:  path,
			width: width,
			doc:   doc,
			err:   err,
		}
	}
}

func previewStatCmd(path string) tea.Cmd {
	return func() tea.Msg {
		info, err := os.Stat(path)
		if err != nil {
			return previewStatMsg{path: path, err: err}
		}
		return previewStatMsg{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime(),
		}
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		cmds := []tea.Cmd{tick()}
		if m.preview.Path != "" {
			cmds = append(cmds, previewStatCmd(m.preview.Path))
		}
		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		prevWidth := m.previewRenderWidth()
		m.width = msg.Width
		m.height = msg.Height
		m.clampPreviewOffset()
		m.clampOutlineCursor()

		nextWidth := m.previewRenderWidth()
		switch {
		case m.pendingOpenPath != "" && nextWidth > 0:
			return m.requestPreview(m.pendingOpenPath)
		case prevWidth != nextWidth && nextWidth > 0:
			if target := m.desiredPreviewPath(); target != "" {
				return m.requestPreview(target)
			}
		default:
			return m, nil
		}

	case dirLoadedMsg:
		if msg.path != m.currentDir || msg.showAll != m.showAllFiles {
			return m, nil
		}
		if msg.err != nil {
			m.status = "Directory error: " + msg.err.Error()
			return m, nil
		}

		m.currentDir = msg.path
		m.allEntries = msg.items
		m.applyBrowserFilter(msg.focus)
		m.status = m.currentEntrySummary()

		if target := m.desiredPreviewPath(); target != "" && filepath.Dir(target) == m.currentDir && m.previewRenderWidth() > 0 {
			return m.requestPreview(target)
		}
		if !m.readerMode {
			m.clearPreview()
		}
		return m, nil

	case previewLoadedMsg:
		if msg.err != nil {
			m.status = "Preview error: " + msg.err.Error()
			return m, nil
		}
		if msg.path != m.activePreviewPath || msg.width != m.activePreviewWidth {
			return m, nil
		}

		if msg.path != m.preview.Path {
			m.searchMatchIndex = 0
		}
		m.preview = msg.doc
		m.previewLines = strings.Split(msg.doc.Content, "\n")
		m.plainPreview = plainLines(m.previewLines)
		m.outline = buildOutline(msg.doc, m.previewLines)
		m.previewOffset = m.previewOffsetForSourceLine(m.pendingSourceLine)
		m.clampOutlineCursor()
		m.rebuildSearchMatches()
		m.pendingOpenPath = ""
		m.pendingSourceLine = 0
		m.status = m.previewSummary()
		m.clampPreviewOffset()
		m.rememberCurrentSourceLine()
		m.syncOutlineWithPreview()
		return m, nil

	case previewStatMsg:
		if msg.path != m.preview.Path {
			return m, nil
		}
		if msg.err != nil {
			if os.IsNotExist(msg.err) {
				m.status = "Preview file removed: " + filepath.Base(msg.path)
				delete(m.previewAnchors, msg.path)
				m.clearPreview()
				return m, loadDirectoryCmd(m.currentDir, currentEntryName(m), m.showAllFiles)
			}
			m.status = "Preview stat error: " + msg.err.Error()
			return m, nil
		}
		if msg.modTime.Equal(m.preview.ModTime) && msg.size == m.preview.Size {
			return m, nil
		}
		m.status = "File updated: " + filepath.Base(msg.path)
		return m, tea.Batch(
			loadDirectoryCmd(m.currentDir, currentEntryName(m), m.showAllFiles),
			m.rerenderPreviewCmd(msg.path),
		)
	}

	if m.browserFilterInputActive {
		return m.updateBrowserFilterInput(msg)
	}
	if m.searchInputActive {
		return m.updateSearchInput(msg)
	}
	if m.outlineView == outlineFull {
		return m.updateFullOutline(msg)
	}

	switch m.focus {
	case focusPreview:
		return m.updatePreview(msg)
	case focusOutline:
		return m.updateSideOutline(msg)
	default:
		return m.updateBrowser(msg)
	}
}

func (m Model) updateBrowser(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.browserFilterInputActive = true
		m.status = "Filter: /" + m.browserFilterQuery
		return m, nil
	case "a":
		return m.toggleShowAllFiles()
	case "esc":
		if m.browserFilterQuery == "" {
			return m, nil
		}
		m.browserFilterQuery = ""
		m.browserFilterInputActive = false
		m.applyBrowserFilter("")
		m.status = "Filter cleared"
		return m.syncHoverPreview()
	case "down", "j":
		m.moveCursor(1)
		return m.syncHoverPreview()
	case "up", "k":
		m.moveCursor(-1)
		return m.syncHoverPreview()
	case "g", "home":
		m.cursor = 0
		m.status = m.currentEntrySummary()
		return m.syncHoverPreview()
	case "G", "end":
		if len(m.entries) > 0 {
			m.cursor = len(m.entries) - 1
		}
		m.status = m.currentEntrySummary()
		return m.syncHoverPreview()
	case "left", "h", "backspace":
		parent := browser.ParentDir(m.currentDir)
		if parent == m.currentDir {
			return m, nil
		}
		return m.enterDirectory(parent, filepath.Base(m.currentDir))
	case "right", "l", "enter":
		entry, ok := m.currentEntry()
		if !ok {
			return m, nil
		}
		switch {
		case entry.IsParent():
			return m.enterDirectory(entry.Path, filepath.Base(m.currentDir))
		case entry.IsDirectory():
			return m.enterDirectory(entry.Path, "")
		case entry.IsViewable():
			m.readerMode = true
			m.zenMode = false
			m.focus = focusPreview
			m.status = m.previewSummary()
			return m.requestPreview(entry.Path)
		default:
			m.status = "Not a viewable file: " + entry.Name
			return m, nil
		}
	case "r":
		return m.refresh()
	default:
		return m, nil
	}
}

func (m Model) updateBrowserFilterInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "esc":
		m.browserFilterInputActive = false
		if m.browserFilterQuery == "" {
			m.status = m.currentEntrySummary()
		} else {
			m.status = m.browserFilterSummary()
		}
		return m, nil
	case "enter":
		m.browserFilterInputActive = false
		if m.browserFilterQuery == "" {
			m.status = m.currentEntrySummary()
		} else {
			m.status = m.browserFilterSummary()
		}
		return m, nil
	case "backspace":
		if m.browserFilterQuery != "" {
			runes := []rune(m.browserFilterQuery)
			m.browserFilterQuery = string(runes[:len(runes)-1])
			m.applyBrowserFilter("")
		}
		return m.syncHoverPreview()
	default:
		if key.Type != tea.KeyRunes {
			return m, nil
		}
		m.browserFilterQuery += string(key.Runes)
		m.applyBrowserFilter("")
		return m.syncHoverPreview()
	}
}

func (m Model) updatePreview(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "a":
		return m.toggleShowAllFiles()
	case "esc":
		return m.leaveReaderMode()
	case "left":
		if m.outlineView == outlineSide && len(m.outline) > 0 {
			m.focus = focusOutline
			m.outlineCursor = m.currentOutlineIndex()
			m.status = fmt.Sprintf("Outline: %s", m.outline[m.outlineCursor].Text)
			return m, nil
		}
		return m.leaveReaderMode()
	case "h":
		return m.leaveReaderMode()
	case "/":
		m.searchInputActive = true
		m.status = "Search: /" + m.searchQuery
		return m, nil
	case "[":
		return m.jumpHeading(-1)
	case "]":
		return m.jumpHeading(1)
	case "-", "_":
		return m.adjustReaderWidth(-readerContentWidthStep)
	case "=", "+":
		return m.adjustReaderWidth(readerContentWidthStep)
	case "n":
		return m.nextSearchMatch(1)
	case "N":
		return m.nextSearchMatch(-1)
	case "o":
		return m.toggleOutlineMode()
	case "z":
		return m.toggleZenMode()
	case "down", "j":
		m.scrollPreview(1)
		return m, nil
	case "up", "k":
		m.scrollPreview(-1)
		return m, nil
	case "pgdown", "f":
		m.scrollPreview(m.previewContentHeight())
		return m, nil
	case "pgup", "b":
		m.scrollPreview(-m.previewContentHeight())
		return m, nil
	case "g", "home":
		m.previewOffset = 0
		m.rememberCurrentSourceLine()
		m.syncOutlineWithPreview()
		return m, nil
	case "G", "end":
		m.previewOffset = max(0, len(m.previewLines)-m.previewContentHeight())
		m.rememberCurrentSourceLine()
		m.syncOutlineWithPreview()
		return m, nil
	case "r":
		return m.refresh()
	default:
		return m, nil
	}
}

func (m Model) updateSideOutline(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "right", "l":
		m.focus = focusPreview
		m.outlineCursor = m.currentOutlineIndex()
		m.status = m.previewSummary()
		return m, nil
	case "esc":
		m.focus = focusPreview
		m.outlineCursor = m.currentOutlineIndex()
		m.status = m.previewSummary()
		return m, nil
	case "left", "h":
		return m.leaveReaderMode()
	case "down", "j":
		m.outlineCursor++
		m.clampOutlineCursor()
		if len(m.outline) > 0 {
			m.status = fmt.Sprintf("Outline: %s", m.outline[m.outlineCursor].Text)
		}
		return m, nil
	case "up", "k":
		m.outlineCursor--
		m.clampOutlineCursor()
		if len(m.outline) > 0 {
			m.status = fmt.Sprintf("Outline: %s", m.outline[m.outlineCursor].Text)
		}
		return m, nil
	case "g", "home":
		m.outlineCursor = 0
		if len(m.outline) > 0 {
			m.status = fmt.Sprintf("Outline: %s", m.outline[m.outlineCursor].Text)
		}
		return m, nil
	case "G", "end":
		if len(m.outline) > 0 {
			m.outlineCursor = len(m.outline) - 1
			m.status = fmt.Sprintf("Outline: %s", m.outline[m.outlineCursor].Text)
		}
		return m, nil
	case "enter":
		if len(m.outline) == 0 {
			return m, nil
		}
		m.jumpToPreviewLine(m.outline[m.outlineCursor].PreviewLine)
		m.focus = focusPreview
		m.outlineCursor = m.currentOutlineIndex()
		m.status = m.previewSummary()
		return m, nil
	case "o":
		return m.toggleOutlineMode()
	case "/":
		m.focus = focusPreview
		m.outlineCursor = m.currentOutlineIndex()
		m.searchInputActive = true
		m.status = "Search: /" + m.searchQuery
		return m, nil
	case "r":
		return m.refresh()
	default:
		return m, nil
	}
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	focusName := ""
	if entry, ok := m.currentEntry(); ok {
		focusName = entry.Name
	}
	cmds := []tea.Cmd{loadDirectoryCmd(m.currentDir, focusName, m.showAllFiles)}
	if m.preview.Path != "" {
		cmds = append(cmds, m.rerenderPreviewCmd(m.preview.Path))
	}
	return m, tea.Batch(cmds...)
}

func (m Model) requestPreview(path string) (tea.Model, tea.Cmd) {
	return m, m.requestPreviewCmd(path)
}

func (m Model) rerenderPreview(path string) (tea.Model, tea.Cmd) {
	return m, m.rerenderPreviewCmd(path)
}

func (m *Model) requestPreviewCmd(path string) tea.Cmd {
	return m.previewCmd(path, false)
}

func (m *Model) rerenderPreviewCmd(path string) tea.Cmd {
	return m.previewCmd(path, true)
}

func (m *Model) previewCmd(path string, rerender bool) tea.Cmd {
	width := m.previewRenderWidth()
	m.pendingOpenPath = path
	if width <= 0 {
		m.status = "Waiting for terminal size to open preview"
		return nil
	}

	if !rerender && m.preview.Path == path && width == m.preview.Width {
		m.pendingOpenPath = ""
		m.status = m.previewSummary()
		return nil
	}

	m.rememberCurrentSourceLine()
	m.pendingSourceLine = m.sourceLineForPath(path)
	m.activePreviewPath = path
	m.activePreviewWidth = width
	return renderPreviewCmd(m.renderer, path, width)
}

func (m Model) enterDirectory(path, focus string) (tea.Model, tea.Cmd) {
	m.currentDir = filepath.Clean(path)
	m.pendingFocusName = focus
	m.previewAnchors = make(map[string]int)
	m.browserFilterQuery = ""
	m.browserFilterInputActive = false
	m.readerMode = false
	m.zenMode = false
	m.outlineView = outlineHidden
	m.searchInputActive = false
	m.focus = focusBrowser
	m.clearPreview()
	m.status = "Directory: " + m.currentDir
	return m, loadDirectoryCmd(m.currentDir, focus, m.showAllFiles)
}

func (m *Model) moveCursor(delta int) {
	if len(m.entries) == 0 {
		return
	}
	m.cursor += delta
	m.ensureCursorInRange()
	m.status = m.currentEntrySummary()
}

func (m *Model) focusEntryByName(name string) {
	for i, entry := range m.entries {
		if entry.Name == name {
			m.cursor = i
			return
		}
	}
}

func (m *Model) applyBrowserFilter(focusName string) {
	if focusName == "" {
		if entry, ok := m.currentEntry(); ok && filepath.Dir(entry.Path) == m.currentDir {
			focusName = entry.Name
		}
	}

	query := strings.ToLower(strings.TrimSpace(m.browserFilterQuery))
	filtered := make([]browser.Entry, 0, len(m.allEntries))
	for _, entry := range m.allEntries {
		if entry.IsParent() {
			filtered = append(filtered, entry)
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(entry.DisplayName()), query) {
			filtered = append(filtered, entry)
		}
	}

	m.entries = filtered
	m.cursor = 0
	if focusName != "" {
		m.focusEntryByName(focusName)
	}
	if len(m.entries) == 0 {
		return
	}
	if focusName == "" || m.entries[m.cursor].Name != focusName {
		m.focusInitialEntry()
	}
	m.ensureCursorInRange()
}

func (m *Model) focusInitialEntry() {
	for i, entry := range m.entries {
		if !entry.IsParent() {
			m.cursor = i
			return
		}
	}
	m.cursor = 0
}

func (m *Model) ensureCursorInRange() {
	if len(m.entries) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
	}
}

func (m *Model) currentEntry() (browser.Entry, bool) {
	if m.cursor < 0 || m.cursor >= len(m.entries) {
		return browser.Entry{}, false
	}
	return m.entries[m.cursor], true
}

func (m *Model) clearPreview() {
	m.preview = render.Document{}
	m.previewLines = nil
	m.plainPreview = nil
	m.previewOffset = 0
	m.outline = nil
	m.outlineView = outlineHidden
	m.outlineCursor = 0
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchMatchIndex = 0
	m.searchInputActive = false
	m.pendingOpenPath = ""
	m.pendingSourceLine = 0
	m.activePreviewPath = ""
	m.activePreviewWidth = 0
}

func (m *Model) rememberCurrentSourceLine() {
	if m.preview.Path == "" {
		return
	}
	if filepath.Dir(m.preview.Path) != m.currentDir {
		return
	}
	if m.previewAnchors == nil {
		m.previewAnchors = make(map[string]int)
	}
	m.previewAnchors[m.preview.Path] = m.currentSourceLine()
}

func (m Model) sourceLineForPath(path string) int {
	if path == "" {
		return 0
	}
	if path == m.preview.Path {
		return m.currentSourceLine()
	}
	if filepath.Dir(path) != m.currentDir {
		return 0
	}
	if m.previewAnchors == nil {
		return 0
	}
	return m.previewAnchors[path]
}

func (m Model) currentSourceLine() int {
	sourceCount := m.preview.LineCount
	if sourceCount <= 0 || len(m.previewLines) == 0 {
		return 0
	}
	if mapped, ok := m.mapPreviewOffsetToSourceLine(m.previewOffset); ok {
		return mapped
	}
	return mapLineBetweenRanges(m.previewOffset, 0, len(m.previewLines), 0, sourceCount)
}

func (m Model) previewOffsetForSourceLine(sourceLine int) int {
	if len(m.previewLines) == 0 {
		return 0
	}
	sourceCount := m.preview.LineCount
	if sourceCount <= 0 {
		return 0
	}
	if sourceLine < 0 {
		sourceLine = 0
	}
	if sourceLine >= sourceCount {
		sourceLine = sourceCount - 1
	}
	if mapped, ok := m.mapSourceLineToPreviewOffset(sourceLine); ok {
		return mapped
	}
	return mapLineBetweenRanges(sourceLine, 0, sourceCount, 0, len(m.previewLines))
}

func (m Model) syncHoverPreview() (tea.Model, tea.Cmd) {
	if m.readerMode {
		return m, nil
	}

	entry, ok := m.currentEntry()
	if !ok || !entry.IsViewable() {
		m.clearPreview()
		m.status = m.currentEntrySummary()
		return m, nil
	}

	return m.requestPreview(entry.Path)
}

func (m Model) toggleShowAllFiles() (tea.Model, tea.Cmd) {
	focusName := ""
	if m.readerMode && m.preview.Path != "" {
		focusName = filepath.Base(m.preview.Path)
	} else if entry, ok := m.currentEntry(); ok {
		focusName = entry.Name
	}

	m.showAllFiles = !m.showAllFiles
	if m.showAllFiles {
		m.status = "All files"
	} else {
		m.status = "Docs only"
	}

	return m, loadDirectoryCmd(m.currentDir, focusName, m.showAllFiles)
}

func (m Model) leaveReaderMode() (tea.Model, tea.Cmd) {
	m.readerMode = false
	m.zenMode = false
	m.outlineView = outlineHidden
	m.searchInputActive = false
	m.focus = focusBrowser
	m.status = m.currentEntrySummary()
	return m.syncHoverPreview()
}

func (m Model) toggleZenMode() (tea.Model, tea.Cmd) {
	if !m.readerMode {
		return m, nil
	}

	m.zenMode = !m.zenMode
	if m.zenMode {
		m.status = "Zen mode"
	} else {
		m.status = m.previewSummary()
	}

	if m.preview.Path == "" {
		return m, nil
	}
	return m.rerenderPreview(m.preview.Path)
}

func (m Model) adjustReaderWidth(delta int) (tea.Model, tea.Cmd) {
	if !m.readerMode || m.preview.Path == "" {
		return m, nil
	}

	innerWidth := m.previewPanelInnerWidth()
	currentWidth := m.readerContentWidth(innerWidth)
	if currentWidth == 0 {
		return m, nil
	}

	minWidth := min(innerWidth, readerMinContentWidth)
	maxWidth := min(innerWidth, readerManualMaxContentWidth)
	nextWidth := currentWidth + delta
	if nextWidth < minWidth {
		nextWidth = minWidth
	}
	if nextWidth > maxWidth {
		nextWidth = maxWidth
	}
	if nextWidth == currentWidth {
		m.status = fmt.Sprintf("Reading width: %d cols", currentWidth)
		return m, nil
	}

	m.readerWidthOffset += nextWidth - currentWidth
	m.status = fmt.Sprintf("Reading width: %d cols", nextWidth)
	return m.rerenderPreview(m.preview.Path)
}

func (m Model) toggleOutlineMode() (tea.Model, tea.Cmd) {
	if m.preview.Path == "" || len(m.outline) == 0 {
		m.status = "No headings found"
		return m, nil
	}

	prevWidth := m.previewRenderWidth()
	switch m.outlineView {
	case outlineHidden:
		m.outlineView = outlineFull
		m.status = fmt.Sprintf("Outline full: %d headings", len(m.outline))
	case outlineFull:
		m.outlineView = outlineSide
		m.focus = focusPreview
		m.status = fmt.Sprintf("Outline side: %d headings", len(m.outline))
	default:
		m.outlineView = outlineHidden
		m.focus = focusPreview
		m.status = m.previewSummary()
	}
	m.searchInputActive = false
	if m.outlineView != outlineHidden {
		m.outlineCursor = m.closestOutlineIndex()
	}
	return m.syncPreviewLayout(prevWidth)
}

func (m Model) desiredPreviewPath() string {
	if m.pendingOpenPath != "" {
		return m.pendingOpenPath
	}
	if m.readerMode && m.preview.Path != "" {
		return m.preview.Path
	}
	entry, ok := m.currentEntry()
	if ok && entry.IsMarkdown() {
		return entry.Path
	}
	return ""
}

func (m *Model) scrollPreview(delta int) {
	m.previewOffset += delta
	m.clampPreviewOffset()
	m.rememberCurrentSourceLine()
	m.syncOutlineWithPreview()
}

func (m *Model) jumpToPreviewLine(line int) {
	m.previewOffset = line
	m.clampPreviewOffset()
	m.rememberCurrentSourceLine()
	m.syncOutlineWithPreview()
}

func (m *Model) clampPreviewOffset() {
	limit := max(0, len(m.previewLines)-m.previewContentHeight())
	if m.previewOffset < 0 {
		m.previewOffset = 0
	}
	if m.previewOffset > limit {
		m.previewOffset = limit
	}
}

func (m *Model) clampOutlineCursor() {
	if len(m.outline) == 0 {
		m.outlineCursor = 0
		return
	}
	if m.outlineCursor < 0 {
		m.outlineCursor = 0
	}
	if m.outlineCursor >= len(m.outline) {
		m.outlineCursor = len(m.outline) - 1
	}
}

func (m *Model) syncOutlineWithPreview() {
	if m.outlineView != outlineSide || len(m.outline) == 0 {
		return
	}
	m.outlineCursor = m.currentOutlineIndex()
}

func (m Model) syncPreviewLayout(previousWidth int) (tea.Model, tea.Cmd) {
	nextWidth := m.previewRenderWidth()
	if m.preview.Path == "" || nextWidth <= 0 || previousWidth == nextWidth {
		return m, nil
	}
	return m.rerenderPreview(m.preview.Path)
}

func (m Model) previewContentHeight() int {
	return max(1, m.panelHeight()-4)
}

func (m Model) panelHeight() int {
	return max(6, m.height-3)
}

func (m Model) previewRenderWidth() int {
	innerWidth := m.previewPanelInnerWidth()
	if innerWidth == 0 {
		return 0
	}
	contentWidth, _ := m.previewBodyLayout(innerWidth)
	return max(20, contentWidth)
}

func (m Model) previewPanelInnerWidth() int {
	if m.width == 0 {
		return 0
	}
	previewWidth := m.width - m.browserWidth()
	return max(20, previewWidth-2)
}

func (m Model) browserWidth() int {
	if m.outlineView == outlineFull {
		return 0
	}
	if m.outlineView == outlineSide {
		return min(max(24, m.width/4), 34)
	}
	if m.zenMode {
		return 0
	}
	if m.readerMode {
		return min(max(22, m.width/5), 30)
	}
	return max(30, m.width*2/5)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}
	if m.width < 40 || m.height < 10 {
		return m.viewTooSmall()
	}
	return m.viewMain()
}

func (m Model) viewTooSmall() string {
	lines := []string{
		titleStyle.Render("skimd"),
		errorStyle.Render(fmt.Sprintf("Terminal too small: %dx%d", m.width, m.height)),
		subtleStyle.Render("Resize the terminal to at least 40x10."),
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewMain() string {
	title := titleStyle.Render("skimd  " + m.currentDir)
	info := m.renderInfoLine()
	help := m.renderHelp()

	panelHeight := m.panelHeight()
	browserWidth := m.browserWidth()
	previewWidth := m.width - browserWidth

	body := ""
	if browserWidth <= 0 {
		body = m.renderPreviewPanel(m.width, panelHeight)
	} else {
		left := m.renderBrowserPanel(browserWidth, panelHeight)
		if m.outlineView == outlineSide {
			left = m.renderOutlinePanel(browserWidth, panelHeight)
		}
		right := m.renderPreviewPanel(previewWidth, panelHeight)
		body = joinHorizontalFixed(left, right)
	}

	return strings.Join([]string{title, info, body, help}, "\n")
}

func (m Model) renderInfoLine() string {
	if m.searchInputActive {
		return subtleStyle.Render(padOrTruncate("Search: /"+m.searchQuery, m.width))
	}
	if m.browserFilterInputActive {
		return subtleStyle.Render(padOrTruncate("Filter: /"+m.browserFilterQuery, m.width))
	}
	if !m.readerMode && m.browserFilterQuery != "" {
		return subtleStyle.Render(padOrTruncate(m.browserFilterSummary(), m.width))
	}
	info := m.status
	if info == "" {
		info = m.currentEntrySummary()
	}
	return subtleStyle.Render(padOrTruncate(info, m.width))
}

func (m Model) renderBrowserPanel(width, height int) string {
	innerWidth := width - 2
	innerHeight := height - 2
	contentHeight := max(1, innerHeight-2)

	header := fmt.Sprintf("[ %s ]  %s", m.browserModeLabel(), m.browserPanelSummary())
	separator := strings.Repeat("─", innerWidth)

	lines := make([]string, 0, innerHeight)
	lines = append(lines, titleStyle.Render(padOrTruncate(header, innerWidth)))
	lines = append(lines, subtleStyle.Render(padOrTruncate(separator, innerWidth)))

	start := scrollWindow(len(m.entries), m.cursor, contentHeight)
	for i := 0; i < contentHeight; i++ {
		idx := start + i
		if idx >= len(m.entries) {
			lines = append(lines, strings.Repeat(" ", innerWidth))
			continue
		}
		lines = append(lines, m.formatEntryRow(m.entries[idx], idx == m.cursor, innerWidth))
	}

	return drawBorder(strings.Join(lines, "\n"), width, innerHeight)
}

func (m Model) renderOutlinePanel(width, height int) string {
	innerWidth := width - 2
	innerHeight := height - 2
	contentHeight := max(1, innerHeight-2)

	header := fmt.Sprintf("[ Outline ]  %d headings", len(m.outline))
	lines := make([]string, 0, innerHeight)
	lines = append(lines, titleStyle.Render(padOrTruncate(header, innerWidth)))
	lines = append(lines, subtleStyle.Render(strings.Repeat("─", innerWidth)))
	lines = append(lines, fixedBox(m.renderOutlineContent(innerWidth, contentHeight), innerWidth, contentHeight))

	return drawBorder(strings.Join(lines, "\n"), width, innerHeight)
}

func (m Model) renderPreviewPanel(width, height int) string {
	innerWidth := width - 2
	innerHeight := height - 2
	contentHeight := max(1, innerHeight-3)
	contentWidth, leftPad := m.previewBodyLayout(innerWidth)

	lines := make([]string, 0, innerHeight)
	lines = append(lines, titleStyle.Render(padOrTruncate(m.previewPanelHeader(), innerWidth)))
	lines = append(lines, subtleStyle.Render(padOrTruncate(m.previewMetaLine(), innerWidth)))
	lines = append(lines, subtleStyle.Render(strings.Repeat("─", innerWidth)))

	if m.preview.Path == "" {
		placeholder := centered("Move to a markdown file to preview it.", innerWidth)
		lines = append(lines, fixedBox(placeholder, innerWidth, contentHeight))
		return drawBorder(strings.Join(lines, "\n"), width, innerHeight)
	}
	if m.outlineView == outlineFull {
		lines = append(lines, fixedBox(m.renderOutlineContent(innerWidth, contentHeight), innerWidth, contentHeight))
		return drawBorder(strings.Join(lines, "\n"), width, innerHeight)
	}

	lines = append(lines, fixedBox(m.renderPreviewBody(contentWidth, leftPad, contentHeight), innerWidth, contentHeight))
	return drawBorder(strings.Join(lines, "\n"), width, innerHeight)
}

func (m Model) formatEntryRow(entry browser.Entry, selected bool, width int) string {
	icon := "·"
	style := lipgloss.NewStyle().Foreground(colorOther)

	switch entry.Kind {
	case browser.EntryParent:
		icon = "↩"
		style = lipgloss.NewStyle().Foreground(colorAccent)
	case browser.EntryDirectory:
		icon = "▸"
		style = lipgloss.NewStyle().Foreground(colorDir)
	case browser.EntryMarkdown:
		icon = "•"
		style = lipgloss.NewStyle().Foreground(colorMarkdown)
	case browser.EntryText:
		icon = "◦"
		style = lipgloss.NewStyle().Foreground(colorMarkdown)
	}

	left := style.Render(fmt.Sprintf(" %s %s", icon, entry.DisplayName()))
	rightText := m.entryMeta(entry)
	row := left
	if rightText != "" && width > 8 {
		right := subtleStyle.Render(rightText)
		maxLeft := max(1, width-ansi.StringWidth(right)-1)
		left = ansi.Truncate(left, maxLeft, "")
		gap := width - ansi.StringWidth(left) - ansi.StringWidth(right)
		if gap < 1 {
			gap = 1
		}
		row = left + strings.Repeat(" ", gap) + right
	}
	row = padOrTruncate(row, width)

	if selected {
		cursor := "›"
		if m.focus == focusPreview {
			cursor = " "
		}
		row = padOrTruncate(cursor+" "+row, width)
		return lipgloss.NewStyle().
			Background(colorSelected).
			Foreground(colorCursor).
			Bold(m.focus == focusBrowser).
			Render(padOrTruncate(row, width))
	}

	return padOrTruncate("  "+row, width)
}

func (m Model) browserModeLabel() string {
	if m.showAllFiles {
		return "Files"
	}
	return "Docs"
}

func (m Model) browserPanelSummary() string {
	directories := 0
	markdown := 0
	other := 0

	for _, entry := range m.entries {
		switch entry.Kind {
		case browser.EntryDirectory:
			directories++
		case browser.EntryMarkdown:
			markdown++
		case browser.EntryOther:
			other++
		}
	}

	if m.showAllFiles {
		return fmt.Sprintf("%d md  %d dirs  %d other", markdown, directories, other)
	}
	return fmt.Sprintf("%d md  %d dirs", markdown, directories)
}

func (m Model) previewPanelHeader() string {
	if m.preview.Path == "" {
		return "[ Preview ]"
	}

	mode := "Preview"
	switch {
	case m.outlineView == outlineFull:
		mode = "Outline"
	case m.zenMode:
		mode = "Zen"
	case m.readerMode:
		mode = "Reading"
	}
	return fmt.Sprintf("[ %s ]  %s", mode, filepath.Base(m.preview.Path))
}

func (m Model) previewMetaLine() string {
	if m.preview.Path == "" {
		if m.showAllFiles {
			return "Browse a markdown file to preview it."
		}
		return "Navigate folders first, then move onto a markdown file to preview it."
	}

	parts := []string{
		humanSize(m.preview.Size),
		fmt.Sprintf("%d lines", m.preview.LineCount),
		fmt.Sprintf("%d headings", len(m.preview.Headings)),
	}
	if m.readerMode || m.zenMode {
		parts = append(parts, fmt.Sprintf("%d cols", m.previewRenderWidth()))
	}
	if m.outlineView == outlineFull {
		parts = append(parts, "full")
	} else if m.outlineView == outlineSide {
		parts = append(parts, "side")
	}
	if section := m.currentSectionLabel(); section != "" {
		parts = append(parts, "§ "+section)
	}
	parts = append(parts, fmt.Sprintf("%s ago", humanAge(m.preview.ModTime)), m.previewScrollLabel())
	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		parts = append(parts, fmt.Sprintf("search %d/%d", m.searchMatchIndex+1, len(m.searchMatches)))
	}
	return strings.Join(parts, "  ")
}

func (m Model) entryMeta(entry browser.Entry) string {
	switch entry.Kind {
	case browser.EntryDirectory:
		return "dir"
	case browser.EntryMarkdown, browser.EntryOther:
		return fmt.Sprintf("%s  %s", humanAge(entry.ModTime), humanSize(entry.Size))
	default:
		return ""
	}
}

func (m Model) currentEntrySummary() string {
	entry, ok := m.currentEntry()
	if !ok {
		if m.browserFilterQuery != "" {
			return m.browserFilterSummary()
		}
		return "No files"
	}

	switch entry.Kind {
	case browser.EntryParent:
		return "Parent directory"
	case browser.EntryDirectory:
		return "Directory: " + entry.Path
	case browser.EntryMarkdown:
		return fmt.Sprintf("Markdown: %s  %s  %s ago", entry.Name, humanSize(entry.Size), humanAge(entry.ModTime))
	default:
		return fmt.Sprintf("Other file: %s  %s  %s ago", entry.Name, humanSize(entry.Size), humanAge(entry.ModTime))
	}
}

func (m Model) browserFilterSummary() string {
	count := 0
	for _, entry := range m.entries {
		if !entry.IsParent() {
			count++
		}
	}
	if count == 0 {
		return "Filter: /" + m.browserFilterQuery + "  no matches"
	}
	return fmt.Sprintf("Filter: /%s  %d match", m.browserFilterQuery, count) + pluralSuffix(count)
}

func (m Model) previewSummary() string {
	if m.preview.Path == "" {
		return "No preview"
	}
	label := "Preview"
	if m.zenMode {
		label = "Zen"
	} else if m.readerMode {
		label = "Open"
	}
	summary := fmt.Sprintf("%s: %s  %s  %d headings  %s ago", label, m.preview.Path, humanSize(m.preview.Size), len(m.preview.Headings), humanAge(m.preview.ModTime))
	if m.preview.Fallback {
		summary += "  raw fallback"
	}
	if m.searchQuery != "" && len(m.searchMatches) > 0 {
		summary += fmt.Sprintf("  search %d/%d", m.searchMatchIndex+1, len(m.searchMatches))
	}
	return summary
}

func (m Model) previewScrollLabel() string {
	sourceCount := m.preview.LineCount
	if sourceCount <= 0 {
		return "0%"
	}
	if sourceCount == 1 {
		return "100%"
	}
	percent := int(float64(m.currentSourceLine()) / float64(sourceCount-1) * 100)
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	return fmt.Sprintf("%d%%", percent)
}

func (m Model) renderHelp() string {
	type helpItem struct {
		key  string
		desc string
	}

	var keys []helpItem
	switch {
	case m.browserFilterInputActive:
		keys = []helpItem{
			{"type", "query"},
			{"Enter", "apply"},
			{"Esc", "close"},
			{"Backspace", "delete"},
		}
	case m.searchInputActive:
		keys = []helpItem{
			{"type", "query"},
			{"Enter", "apply"},
			{"Esc", "cancel"},
			{"Backspace", "delete"},
		}
	case m.outlineView == outlineFull:
		keys = []helpItem{
			{"j/k", "move"},
			{"Enter", "jump"},
			{"o", "side outline"},
			{"Esc", "close"},
			{"q", "quit"},
		}
	case m.outlineView == outlineSide && m.focus == focusOutline:
		keys = []helpItem{
			{"j/k", "move"},
			{"Enter", "jump"},
			{"→", "read"},
			{"Esc", "read"},
		}
		if m.width >= 110 {
			keys = []helpItem{
				{"j/k", "move"},
				{"Enter", "jump"},
				{"→", "read"},
				{"h", "docs"},
				{"Esc", "read"},
			}
		}
	case m.readerMode:
		keys = []helpItem{
			{"j/k", "scroll"},
			{"/", "search"},
			{"[", "prev"},
			{"]", "next"},
			{"Esc", "back"},
		}
		if m.width >= 90 {
			keys = []helpItem{
				{"j/k", "scroll"},
				{"←", "outline"},
				{"/", "search"},
				{"o", "outline"},
				{"[", "prev"},
				{"]", "next"},
				{"Esc", "back"},
			}
		}
		if m.width >= 110 {
			keys = []helpItem{
				{"j/k", "scroll"},
				{"←", "outline"},
				{"/", "search"},
				{"o", "outline"},
				{"[", "prev"},
				{"]", "next"},
				{"- =", "width"},
				{"Esc", "back"},
			}
		}
	default:
		toggleLabel := "all files"
		if m.showAllFiles {
			toggleLabel = "docs only"
		}
		keys = []helpItem{
			{"j/k", "move"},
			{"/", "filter"},
			{"Enter", "open"},
			{"h", "up"},
			{"a", toggleLabel},
			{"q", "quit"},
		}
	}

	parts := make([]string, 0, len(keys))
	for _, item := range keys {
		parts = append(parts, helpKeyStyle.Render(item.key)+" "+subtleStyle.Render(item.desc))
	}
	return padOrTruncate(strings.Join(parts, subtleStyle.Render("  •  ")), m.width)
}

func scrollWindow(total, cursor, height int) int {
	if total <= height || height <= 0 {
		return 0
	}
	start := cursor - height/2
	if start < 0 {
		start = 0
	}
	limit := total - height
	if start > limit {
		start = limit
	}
	return start
}

func centered(text string, width int) string {
	if width <= 0 {
		return ""
	}
	padding := max(0, (width-len(text))/2)
	return strings.Repeat(" ", padding) + text
}

func humanSize(size int64) string {
	switch {
	case size >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(size)/(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(size)/(1<<10))
	default:
		return fmt.Sprintf("%dB", size)
	}
}

func humanAge(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	age := time.Since(t)
	if age < 0 {
		age = 0
	}

	switch {
	case age < time.Minute:
		return fmt.Sprintf("%ds", int(age/time.Second))
	case age < time.Hour:
		return fmt.Sprintf("%dm", int(age/time.Minute))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh", int(age/time.Hour))
	default:
		return fmt.Sprintf("%dd", int(age/(24*time.Hour)))
	}
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "es"
}

func (m Model) previewBodyLayout(innerWidth int) (contentWidth, leftPad int) {
	if innerWidth <= 0 {
		return 0, 0
	}

	limit := previewMaxContentWidth
	if m.readerMode || m.zenMode {
		limit = m.readerContentWidth(innerWidth)
	}
	contentWidth = min(innerWidth, limit)
	leftPad = max(0, (innerWidth-contentWidth)/2)
	return contentWidth, leftPad
}

func (m Model) readerContentWidth(innerWidth int) int {
	if innerWidth <= 0 {
		return 0
	}

	defaultWidth := m.defaultReaderContentWidth(innerWidth)
	minWidth := min(innerWidth, readerMinContentWidth)
	maxWidth := min(innerWidth, readerManualMaxContentWidth)

	width := defaultWidth + m.readerWidthOffset
	if width < minWidth {
		width = minWidth
	}
	if width > maxWidth {
		width = maxWidth
	}
	return width
}

func (m Model) defaultReaderContentWidth(innerWidth int) int {
	if innerWidth <= 0 {
		return 0
	}

	width := innerWidth - 8
	if width < readerBaseContentWidth {
		width = readerBaseContentWidth
	}
	if width > readerAutoMaxContentWidth {
		width = readerAutoMaxContentWidth
	}
	if width > innerWidth {
		width = innerWidth
	}
	return width
}

func (m Model) renderPreviewBody(contentWidth, leftPad, contentHeight int) string {
	if contentWidth <= 0 || contentHeight <= 0 {
		return ""
	}

	visible := m.previewLines
	if m.previewOffset < len(m.previewLines) {
		visible = m.previewLines[m.previewOffset:]
	} else {
		visible = nil
	}

	limit := min(contentHeight, len(visible))
	lines := make([]string, 0, limit)

	for i := 0; i < limit; i++ {
		lineNumber := m.previewOffset + i
		line := visible[i]
		prefix, lineWidth := m.previewLinePrefix(lineNumber, leftPad, contentWidth)
		if m.isCurrentSearchMatchLine(lineNumber) && lineNumber < len(m.plainPreview) && m.searchQuery != "" {
			line = highlightSearchText(m.plainPreview[lineNumber], m.searchQuery)
		}
		if line == "" {
			lines = append(lines, prefix)
			continue
		}
		lines = append(lines, prefix+padOrTruncate(line, lineWidth))
	}

	return strings.Join(lines, "\n")
}

func (m Model) previewLinePrefix(lineNumber, leftPad, contentWidth int) (string, int) {
	prefix := strings.Repeat(" ", leftPad)
	if !m.isCurrentSearchMatchLine(lineNumber) {
		return prefix, contentWidth
	}

	marker := searchMarkerStyle.Render("▎")
	return prefix + marker + " ", max(1, contentWidth-2)
}

func (m Model) currentSectionLabel() string {
	if len(m.outline) == 0 {
		return ""
	}

	current := m.outline[m.currentOutlineIndex()]
	return current.Text
}

func (m Model) mapPreviewOffsetToSourceLine(previewLine int) (int, bool) {
	if len(m.outline) == 0 || len(m.outline) != len(m.preview.Headings) || len(m.previewLines) == 0 || m.preview.LineCount <= 0 {
		return 0, false
	}

	section := m.sectionIndexForPreviewLine(previewLine)
	if section < 0 {
		return 0, false
	}

	sourceStart, sourceEnd, previewStart, previewEnd := m.sectionRanges(section)
	return mapLineBetweenRanges(previewLine, previewStart, previewEnd, sourceStart, sourceEnd), true
}

func (m Model) mapSourceLineToPreviewOffset(sourceLine int) (int, bool) {
	if len(m.outline) == 0 || len(m.outline) != len(m.preview.Headings) || len(m.previewLines) == 0 || m.preview.LineCount <= 0 {
		return 0, false
	}

	section := m.sectionIndexForSourceLine(sourceLine)
	if section < 0 {
		return 0, false
	}

	sourceStart, sourceEnd, previewStart, previewEnd := m.sectionRanges(section)
	return mapLineBetweenRanges(sourceLine, sourceStart, sourceEnd, previewStart, previewEnd), true
}

func (m Model) sectionIndexForPreviewLine(previewLine int) int {
	if len(m.outline) == 0 || previewLine < m.outline[0].PreviewLine {
		return -1
	}

	best := 0
	for i, entry := range m.outline {
		if entry.PreviewLine <= previewLine {
			best = i
			continue
		}
		break
	}
	return best
}

func (m Model) sectionIndexForSourceLine(sourceLine int) int {
	if len(m.preview.Headings) == 0 || sourceLine < m.preview.Headings[0].SourceLine {
		return -1
	}

	best := 0
	for i, heading := range m.preview.Headings {
		if heading.SourceLine <= sourceLine {
			best = i
			continue
		}
		break
	}
	return best
}

func (m Model) sectionRanges(index int) (sourceStart, sourceEnd, previewStart, previewEnd int) {
	sourceCount := max(1, m.preview.LineCount)
	previewCount := max(1, len(m.previewLines))

	sourceStart = clampRangeStart(m.preview.Headings[index].SourceLine, sourceCount)
	previewStart = clampRangeStart(m.outline[index].PreviewLine, previewCount)
	sourceEnd = sourceCount
	previewEnd = previewCount

	if index+1 < len(m.preview.Headings) {
		sourceEnd = clampRangeEnd(m.preview.Headings[index+1].SourceLine, sourceStart, sourceCount)
		previewEnd = clampRangeEnd(m.outline[index+1].PreviewLine, previewStart, previewCount)
	}

	return sourceStart, sourceEnd, previewStart, previewEnd
}

func clampRangeStart(line, total int) int {
	if total <= 0 {
		return 0
	}
	if line < 0 {
		return 0
	}
	if line >= total {
		return total - 1
	}
	return line
}

func clampRangeEnd(line, start, total int) int {
	if total <= 0 {
		return 0
	}
	if line <= start {
		if start+1 > total {
			return total
		}
		return start + 1
	}
	if line > total {
		return total
	}
	return line
}

func mapLineBetweenRanges(line, fromStart, fromEnd, toStart, toEnd int) int {
	fromSpan := max(1, fromEnd-fromStart)
	toSpan := max(1, toEnd-toStart)

	if line < fromStart {
		line = fromStart
	}
	if line >= fromEnd {
		line = fromEnd - 1
	}
	if toSpan == 1 || fromSpan == 1 {
		return toStart
	}

	rel := line - fromStart
	numerator := rel * (toSpan - 1)
	target := toStart + (numerator+(fromSpan-1)/2)/(fromSpan-1)
	if target < toStart {
		return toStart
	}
	if target >= toEnd {
		return toEnd - 1
	}
	return target
}

func (m Model) isCurrentSearchMatchLine(line int) bool {
	if !(m.readerMode || m.zenMode) {
		return false
	}
	if len(m.searchMatches) == 0 || m.searchMatchIndex < 0 || m.searchMatchIndex >= len(m.searchMatches) {
		return false
	}
	return m.searchMatches[m.searchMatchIndex] == line
}

func (m Model) currentOutlineIndex() int {
	if len(m.outline) == 0 || len(m.preview.Headings) != len(m.outline) {
		return m.closestOutlineIndex()
	}

	current := m.currentSourceLine()
	best := 0
	for i, heading := range m.preview.Headings {
		if heading.SourceLine <= current {
			best = i
			continue
		}
		break
	}
	return best
}

func currentEntryName(m Model) string {
	entry, ok := m.currentEntry()
	if !ok {
		return ""
	}
	return entry.Name
}

func (m Model) updateSearchInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "esc":
		m.searchInputActive = false
		m.status = m.previewSummary()
		return m, nil
	case "enter":
		m.searchInputActive = false
		m.rebuildSearchMatches()
		if len(m.searchMatches) == 0 {
			m.status = "No matches: " + m.searchQuery
			return m, nil
		}
		m.searchMatchIndex = 0
		m.jumpToPreviewLine(m.searchMatches[0])
		m.status = m.previewSummary()
		return m, nil
	case "backspace":
		if m.searchQuery != "" {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
		}
		return m, nil
	default:
		if key.Type == tea.KeyRunes {
			m.searchQuery += string(key.Runes)
		}
		return m, nil
	}
}

// Full outline is modal. Side outline stays passive and follows the reader pane.
func (m Model) updateFullOutline(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.String() {
	case "o":
		return m.toggleOutlineMode()
	case "esc":
		prevWidth := m.previewRenderWidth()
		m.outlineView = outlineHidden
		m.focus = focusPreview
		m.status = m.previewSummary()
		return m.syncPreviewLayout(prevWidth)
	case "q", "ctrl+c":
		return m, tea.Quit
	case "down", "j":
		m.outlineCursor++
		m.clampOutlineCursor()
		return m, nil
	case "up", "k":
		m.outlineCursor--
		m.clampOutlineCursor()
		return m, nil
	case "g", "home":
		m.outlineCursor = 0
		return m, nil
	case "G", "end":
		if len(m.outline) > 0 {
			m.outlineCursor = len(m.outline) - 1
		}
		return m, nil
	case "enter":
		if len(m.outline) == 0 {
			return m, nil
		}
		m.jumpToPreviewLine(m.outline[m.outlineCursor].PreviewLine)
		prevWidth := m.previewRenderWidth()
		m.outlineView = outlineHidden
		m.focus = focusPreview
		m.status = m.previewSummary()
		return m.syncPreviewLayout(prevWidth)
	default:
		return m, nil
	}
}

func (m *Model) rebuildSearchMatches() {
	m.searchMatches = findSearchMatches(m.plainPreview, m.searchQuery)
	if len(m.searchMatches) == 0 {
		m.searchMatchIndex = 0
		return
	}
	if m.searchMatchIndex >= len(m.searchMatches) {
		m.searchMatchIndex = len(m.searchMatches) - 1
	}
	if m.searchMatchIndex < 0 {
		m.searchMatchIndex = 0
	}
}

func (m Model) nextSearchMatch(delta int) (tea.Model, tea.Cmd) {
	if len(m.searchMatches) == 0 {
		if m.searchQuery == "" {
			m.status = "Search not set"
		} else {
			m.status = "No matches: " + m.searchQuery
		}
		return m, nil
	}

	total := len(m.searchMatches)
	m.searchMatchIndex = (m.searchMatchIndex + delta + total) % total
	m.jumpToPreviewLine(m.searchMatches[m.searchMatchIndex])
	m.status = m.previewSummary()
	return m, nil
}

func (m Model) jumpHeading(delta int) (tea.Model, tea.Cmd) {
	if len(m.outline) == 0 {
		m.status = "No headings found"
		return m, nil
	}

	target := m.currentOutlineIndex() + delta
	if target < 0 {
		target = 0
	}
	if target >= len(m.outline) {
		target = len(m.outline) - 1
	}

	m.jumpToPreviewLine(m.outline[target].PreviewLine)
	m.status = m.previewSummary()
	return m, nil
}

func (m Model) closestOutlineIndex() int {
	if len(m.outline) == 0 {
		return 0
	}
	best := 0
	for i, entry := range m.outline {
		if entry.PreviewLine <= m.previewOffset {
			best = i
			continue
		}
		break
	}
	return best
}

func (m Model) renderOutlineContent(width, height int) string {
	lines := make([]string, height)
	start := scrollWindow(len(m.outline), m.outlineCursor, height)
	currentIndex := m.currentOutlineIndex()
	hasSelection := m.outlineView == outlineFull || (m.outlineView == outlineSide && m.focus == focusOutline)
	for i := 0; i < height; i++ {
		idx := start + i
		if idx >= len(m.outline) {
			lines[i] = strings.Repeat(" ", width)
			continue
		}

		entry := m.outline[idx]
		indent := strings.Repeat("  ", max(0, entry.Level-1))
		marker := "  "
		rowStyle := lipgloss.NewStyle()
		switch {
		case idx == m.outlineCursor && hasSelection:
			marker = "› "
			rowStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(colorCursor).
				Bold(true)
		case idx == currentIndex:
			marker = "• "
			rowStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)
		}

		row := marker + indent + entry.Text
		row = padOrTruncate(row, width)
		if (idx == m.outlineCursor && hasSelection) || idx == currentIndex {
			lines[i] = rowStyle.Render(padOrTruncate(row, width))
		} else {
			lines[i] = row
		}
	}
	return strings.Join(lines, "\n")
}
