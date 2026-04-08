package ui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/lunemis/skimd/internal/render"
)

func TestFindSearchMatches(t *testing.T) {
	lines := []string{
		"alpha",
		"beta",
		"Alpha beta",
		"gamma",
	}

	got := findSearchMatches(lines, "alpha")
	want := []int{0, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected matches %v, got %v", want, got)
	}
}

func TestBuildOutline(t *testing.T) {
	doc := render.Document{
		Headings: []render.Heading{
			{Level: 1, Text: "Intro", SourceLine: 0},
			{Level: 2, Text: "Details", SourceLine: 2},
		},
	}
	previewLines := []string{
		"Intro",
		"body",
		"Details",
		"more",
	}

	got := buildOutline(doc, previewLines)
	if len(got) != 2 {
		t.Fatalf("expected 2 outline entries, got %d", len(got))
	}
	if got[0].PreviewLine != 0 || got[1].PreviewLine != 2 {
		t.Fatalf("unexpected preview lines: %+v", got)
	}
	if got[0].SourceLine != 0 || got[1].SourceLine != 2 {
		t.Fatalf("unexpected source lines: %+v", got)
	}
}

func TestFindSearchMatchRangesIsCaseInsensitive(t *testing.T) {
	got := findSearchMatchRanges("Alpha beta ALPHA", "alpha")
	want := []searchMatchRange{
		{Start: 0, End: 5},
		{Start: 11, End: 16},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected ranges %+v, got %+v", want, got)
	}
}

func TestHighlightSearchTextHighlightsMatches(t *testing.T) {
	got := highlightSearchText("mode and Mode", "mode")
	if stripped := ansi.Strip(got); stripped != "mode and Mode" {
		t.Fatalf("expected stripped output to preserve text, got %q", stripped)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected styled highlight output, got %q", got)
	}
}
