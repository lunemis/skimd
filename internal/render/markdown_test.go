package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderFileRendersMarkdown(t *testing.T) {
	renderer := NewRenderer(Options{Style: "ascii"})
	path := filepath.Join("..", "..", "testdata", "docs", "simple.md")

	doc, err := renderer.RenderFile(path, 72)
	if err != nil {
		t.Fatalf("RenderFile() error = %v", err)
	}

	plain := ansi.Strip(doc.Content)
	if !strings.Contains(plain, "Simple Document") {
		t.Fatalf("expected heading in rendered output, got %q", plain)
	}
	if !strings.Contains(plain, "first item") {
		t.Fatalf("expected list item in rendered output, got %q", plain)
	}
	if doc.Fallback {
		t.Fatalf("expected markdown renderer to succeed without fallback")
	}
}

func TestRenderFileInvalidatesCacheByWidthAndModTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.md")
	if err := os.WriteFile(path, []byte("# first\n\nbody"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	renderer := NewRenderer(Options{Style: "ascii"})

	doc1, err := renderer.RenderFile(path, 60)
	if err != nil {
		t.Fatalf("RenderFile() error = %v", err)
	}

	doc2, err := renderer.RenderFile(path, 80)
	if err != nil {
		t.Fatalf("RenderFile() second width error = %v", err)
	}
	if doc1.Width == doc2.Width {
		t.Fatalf("expected width-specific cache entries")
	}

	if err := os.WriteFile(path, []byte("# second\n\nbody"), 0o644); err != nil {
		t.Fatalf("WriteFile() update error = %v", err)
	}
	now := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, now, now); err != nil {
		t.Fatalf("Chtimes() error = %v", err)
	}

	doc3, err := renderer.RenderFile(path, 60)
	if err != nil {
		t.Fatalf("RenderFile() after mutation error = %v", err)
	}

	if ansi.Strip(doc1.Content) == ansi.Strip(doc3.Content) {
		t.Fatalf("expected cache invalidation after file change")
	}
}

func TestRenderFileExtractsHeadings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "headings.md")
	content := "# Intro\n\ntext\n\n## Details\n\nmore\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	renderer := NewRenderer(Options{Style: "ascii"})
	doc, err := renderer.RenderFile(path, 72)
	if err != nil {
		t.Fatalf("RenderFile() error = %v", err)
	}

	if len(doc.Headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(doc.Headings))
	}
	if doc.Headings[0].Text != "Intro" || doc.Headings[1].Text != "Details" {
		t.Fatalf("unexpected headings: %+v", doc.Headings)
	}
	if doc.Headings[0].SourceLine != 0 || doc.Headings[1].SourceLine != 4 {
		t.Fatalf("unexpected heading source lines: %+v", doc.Headings)
	}
}

func TestRenderFileLineCountUsesSourceLinesAcrossWidths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lines.md")
	content := "# Intro\n\n" + strings.Repeat("very long wrapped line for width checks\n", 20)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	renderer := NewRenderer(Options{})
	docNarrow, err := renderer.RenderFile(path, 60)
	if err != nil {
		t.Fatalf("RenderFile(narrow) error = %v", err)
	}
	docWide, err := renderer.RenderFile(path, 120)
	if err != nil {
		t.Fatalf("RenderFile(wide) error = %v", err)
	}

	if docNarrow.LineCount != 22 || docWide.LineCount != 22 {
		t.Fatalf("expected source line count 22 across widths, got narrow=%d wide=%d", docNarrow.LineCount, docWide.LineCount)
	}
	if docNarrow.LineCount != docWide.LineCount {
		t.Fatalf("expected width-independent line count, got narrow=%d wide=%d", docNarrow.LineCount, docWide.LineCount)
	}
}

func TestRenderFileUsesReaderTableSeparators(t *testing.T) {
	renderer := NewRenderer(Options{})
	path := filepath.Join("..", "..", "testdata", "docs", "table.md")

	doc, err := renderer.RenderFile(path, 72)
	if err != nil {
		t.Fatalf("RenderFile() error = %v", err)
	}

	plain := ansi.Strip(doc.Content)
	if !strings.Contains(plain, "│") {
		t.Fatalf("expected table column separator in rendered output, got %q", plain)
	}
	if !strings.Contains(plain, "─") {
		t.Fatalf("expected table row separator in rendered output, got %q", plain)
	}
}

func TestPrepareRenderSourceAddsPlaintextToUnlabeledFences(t *testing.T) {
	source := strings.Join([]string{
		"# Intro",
		"",
		"```",
		"plain text block",
		"```",
		"",
		"~~~",
		"another plain block",
		"~~~",
		"",
		"```json",
		`{"ok": true}`,
		"```",
	}, "\n")

	prepared := prepareRenderSource(source)

	if !strings.Contains(prepared, "```plaintext\nplain text block\n```") {
		t.Fatalf("expected unlabeled backtick fence to become plaintext, got %q", prepared)
	}
	if !strings.Contains(prepared, "~~~plaintext\nanother plain block\n~~~") {
		t.Fatalf("expected unlabeled tilde fence to become plaintext, got %q", prepared)
	}
	if !strings.Contains(prepared, "```json\n{\"ok\": true}\n```") {
		t.Fatalf("expected labeled fence to remain unchanged, got %q", prepared)
	}
}

func TestRenderFileDoesNotAnalyseUnlabeledFencedCodeBlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sequence.md")
	lines := []string{
		"1. 사용자 → Portal (브라우저)",
		"2. Portal → ADFS 로그인 페이지 (response_type=id_token, nonce, state)",
		"7. Auth Service:",
		"   b. 클레임 추출: 사용자 식별 claim(`sub`/`email`), `department`, `name`",
		"9. Portal: 필요 시 GET /user/profile로 authoritative 사용자 정보 조회",
	}
	content := strings.Join([]string{
		"```",
		strings.Join(lines, "\n"),
		"```",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	renderer := NewRenderer(Options{})
	doc, err := renderer.RenderFile(path, 120)
	if err != nil {
		t.Fatalf("RenderFile() error = %v", err)
	}

	for _, line := range lines {
		if !strings.Contains(doc.Content, line) {
			t.Fatalf("expected unlabeled code block line to remain contiguous in ANSI output, missing %q in %q", line, doc.Content)
		}
	}
}
