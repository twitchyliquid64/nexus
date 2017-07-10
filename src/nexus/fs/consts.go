package fs

import "nexus/data/fs"

// item kinds
const (
	KindUnknown int = -1
	KindRoot    int = iota
	KindFile
	KindDirectory
)

func miniFSKindToItemKind(miniFSKind int) int {
	switch miniFSKind {
	case fs.FSKindFile:
		return KindFile
	case fs.FSKindDirectory:
		return KindDirectory
	}
	return KindUnknown
}
