package fs

import (
	"context"
	"errors"
	"nexus/data/fs"
	"strings"
)

// ErrHasFiles is returned if one attempts to delete a non-empty directory.
var ErrHasFiles = errors.New("Cannot delete non-empty directory")

func deleteFromSource(ctx context.Context, source *fs.Source, p string, userID int) error {
	src, err := ExpandSource(source)
	if err != nil {
		return err
	}
	return src.Delete(ctx, p, userID)
}

// Delete removes a file from the filesystem, throwing an error if it doesnt exist.
func Delete(ctx context.Context, p string, userID int) error {
	var err error
	if p, err = validatePath(p); err != nil {
		return err
	}

	// identify the source and query that
	sources, err := getSourcesForUser(ctx, userID)
	if err != nil {
		return err
	}

	splitPath := strings.Split(p, "/")
	for _, source := range sources {
		if splitPath[1] == source.Prefix {
			return deleteFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID)
		}
	}
	return errors.New("No such root source")
}
