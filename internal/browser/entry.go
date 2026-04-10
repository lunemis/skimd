package browser

import "time"

type EntryKind int

const (
	EntryParent EntryKind = iota
	EntryDirectory
	EntryMarkdown
	EntryText
	EntryOther
)

type Entry struct {
	Name    string
	Path    string
	Kind    EntryKind
	Size    int64
	ModTime time.Time
}

func (e Entry) IsParent() bool {
	return e.Kind == EntryParent
}

func (e Entry) IsDirectory() bool {
	return e.Kind == EntryDirectory || e.Kind == EntryParent
}

func (e Entry) IsMarkdown() bool {
	return e.Kind == EntryMarkdown
}

func (e Entry) IsText() bool {
	return e.Kind == EntryText
}

func (e Entry) IsViewable() bool {
	return e.Kind == EntryMarkdown || e.Kind == EntryText
}

func (e Entry) DisplayName() string {
	switch e.Kind {
	case EntryParent:
		return ".."
	case EntryDirectory:
		return e.Name + "/"
	default:
		return e.Name
	}
}

type StartLocation struct {
	Dir      string
	Focus    string
	OpenFile string
}
