package fs

import (
	"context"
	"nexus/data/fs"
	"os"
	"strings"
	"time"
)

// ListResultItem represents a single file returned in a List request.
type ListResultItem struct {
	Name     string
	ItemKind int
	Modified time.Time

	// may be set for roots
	SourceDetail int
}

func listSources(ctx context.Context, path string, userID int) ([]ListResultItem, error) {
	sources, err := fs.GetSourcesForUser(ctx, userID, db)
	if err != nil {
		return nil, err
	}

	output := []ListResultItem{
		ListResultItem{
			Name:         "/minifs",
			ItemKind:     KindRoot,
			SourceDetail: fs.FSSourceMiniFS,
		},
	}

	for _, source := range sources {
		output = append(output, ListResultItem{
			Name:         "/" + source.Prefix,
			ItemKind:     KindRoot,
			SourceDetail: source.Kind,
		})
	}

	return output, nil
}

func listFromSource(ctx context.Context, source *fs.Source, path string, userID int) ([]ListResultItem, error) {
	src, err := expandSource(source)
	if err != nil {
		return nil, err
	}
	return src.List(ctx, path, userID)
}

// List returns a list of files at the specified path for the specified user.
func List(ctx context.Context, path string, userID int) ([]ListResultItem, error) {
	var err error
	if path, err = validatePath(path); err != nil {
		return nil, err
	}

	if path == "/" { //special case - list root sources
		return listSources(ctx, path, userID)
	}

	// identify the source and query that
	sources, err := getSourcesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	splitPath := strings.Split(path, "/")
	for _, source := range sources {
		if splitPath[1] == source.Prefix {
			return listFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID)
		}
	}
	return nil, os.ErrNotExist
}
