package browser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadDirectorySortsAndClassifies(t *testing.T) {
	root := t.TempDir()

	mustMkdir(t, filepath.Join(root, "notes"))
	readmePath := filepath.Join(root, "README.md")
	guidePath := filepath.Join(root, "guide.markdown")
	mustWriteFile(t, readmePath, "# hello")
	mustWriteFile(t, guidePath, "# guide")
	mustWriteFile(t, filepath.Join(root, "plain.txt"), "x")

	old := time.Now().Add(-2 * time.Hour)
	newer := time.Now().Add(-5 * time.Minute)
	if err := os.Chtimes(readmePath, old, old); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", readmePath, err)
	}
	if err := os.Chtimes(guidePath, newer, newer); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", guidePath, err)
	}

	entries, err := ReadDirectory(root, ReadOptions{})
	if err != nil {
		t.Fatalf("ReadDirectory() error = %v", err)
	}

	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries, got %d", len(entries))
	}

	offset := 0
	if entries[0].IsParent() {
		offset = 1
	}

	if got := entries[offset].Kind; got != EntryDirectory {
		t.Fatalf("expected first real entry to be directory, got %v", got)
	}

	if got := entries[offset+1].Kind; got != EntryMarkdown {
		t.Fatalf("expected second real entry to be markdown, got %v", got)
	}
	if got := entries[offset+1].Name; got != "guide.markdown" {
		t.Fatalf("expected most recent markdown first, got %q", got)
	}
	if got := entries[offset+2].Name; got != "README.md" {
		t.Fatalf("expected older markdown second, got %q", got)
	}
}

func TestReadDirectoryShowAllIncludesOtherFiles(t *testing.T) {
	root := t.TempDir()

	mustWriteFile(t, filepath.Join(root, "doc.md"), "# hello")
	mustWriteFile(t, filepath.Join(root, "plain.txt"), "x")

	entries, err := ReadDirectory(root, ReadOptions{ShowAllFiles: true})
	if err != nil {
		t.Fatalf("ReadDirectory() error = %v", err)
	}

	foundText := false
	for _, entry := range entries {
		if entry.Kind == EntryText && entry.Name == "plain.txt" {
			foundText = true
			break
		}
	}
	if !foundText {
		t.Fatalf("expected text file to be included when ShowAllFiles is true")
	}
}

func TestResolveStartLocationForDirectoryAndFile(t *testing.T) {
	root := t.TempDir()
	doc := filepath.Join(root, "doc.md")
	mustWriteFile(t, doc, "# doc")

	loc, err := ResolveStartLocation(root)
	if err != nil {
		t.Fatalf("ResolveStartLocation(dir) error = %v", err)
	}
	if loc.Dir != filepath.Clean(root) {
		t.Fatalf("expected dir %q, got %q", root, loc.Dir)
	}
	if loc.OpenFile != "" {
		t.Fatalf("expected no file to open for directory input")
	}

	loc, err = ResolveStartLocation(doc)
	if err != nil {
		t.Fatalf("ResolveStartLocation(file) error = %v", err)
	}
	if loc.Dir != filepath.Dir(doc) {
		t.Fatalf("expected parent dir %q, got %q", filepath.Dir(doc), loc.Dir)
	}
	if loc.Focus != "doc.md" {
		t.Fatalf("expected focus doc.md, got %q", loc.Focus)
	}
	if loc.OpenFile != doc {
		t.Fatalf("expected open file %q, got %q", doc, loc.OpenFile)
	}
}

func TestParentDirStopsAtRoot(t *testing.T) {
	if got := ParentDir("/"); got != "/" {
		t.Fatalf("expected root parent to stay root, got %q", got)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
