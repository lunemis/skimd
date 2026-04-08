package browser

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var markdownExtensions = map[string]struct{}{
	".md":       {},
	".markdown": {},
	".mdown":    {},
	".mkd":      {},
}

type ReadOptions struct {
	ShowAllFiles bool
}

func IsMarkdownFile(name string) bool {
	_, ok := markdownExtensions[strings.ToLower(filepath.Ext(name))]
	return ok
}

func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func ResolveStartLocation(input string) (StartLocation, error) {
	target := input
	if target == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return StartLocation{}, err
		}
		return StartLocation{Dir: filepath.Clean(cwd)}, nil
	}

	expanded, err := ExpandPath(target)
	if err != nil {
		return StartLocation{}, err
	}

	absolute, err := filepath.Abs(expanded)
	if err != nil {
		return StartLocation{}, err
	}

	info, err := os.Stat(absolute)
	if err != nil {
		return StartLocation{}, err
	}

	if info.IsDir() {
		return StartLocation{Dir: filepath.Clean(absolute)}, nil
	}

	return StartLocation{
		Dir:      filepath.Dir(absolute),
		Focus:    filepath.Base(absolute),
		OpenFile: absolute,
	}, nil
}

func ParentDir(path string) string {
	clean := filepath.Clean(path)
	parent := filepath.Dir(clean)
	if parent == "." {
		return clean
	}
	return parent
}

func ReadDirectory(dir string, options ReadOptions) ([]Entry, error) {
	clean := filepath.Clean(dir)
	entries, err := os.ReadDir(clean)
	if err != nil {
		return nil, fmt.Errorf("read directory %q: %w", clean, err)
	}

	result := make([]Entry, 0, len(entries)+1)
	if parent := ParentDir(clean); parent != clean {
		result = append(result, Entry{
			Name: "..",
			Path: parent,
			Kind: EntryParent,
		})
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		item := Entry{
			Name:    entry.Name(),
			Path:    filepath.Join(clean, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}

		switch {
		case entry.IsDir():
			item.Kind = EntryDirectory
		case IsMarkdownFile(entry.Name()):
			item.Kind = EntryMarkdown
		default:
			if !options.ShowAllFiles {
				continue
			}
			item.Kind = EntryOther
		}

		result = append(result, item)
	}

	slices.SortStableFunc(result, func(a, b Entry) int {
		ar := sortRank(a.Kind)
		br := sortRank(b.Kind)
		if ar != br {
			return ar - br
		}
		if a.Kind == EntryMarkdown {
			switch {
			case a.ModTime.After(b.ModTime):
				return -1
			case a.ModTime.Before(b.ModTime):
				return 1
			}
		}
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return result, nil
}

func sortRank(kind EntryKind) int {
	switch kind {
	case EntryParent:
		return 0
	case EntryDirectory:
		return 1
	case EntryMarkdown:
		return 2
	default:
		return 3
	}
}
