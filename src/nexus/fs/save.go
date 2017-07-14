package fs

import (
	"context"
	"errors"
	"nexus/data/fs"
	"strings"
)

func saveFromSource(ctx context.Context, source *fs.Source, path string, userID int, data []byte) error {
	src, err := expandSource(source)
	if err != nil {
		return err
	}
	return src.Save(ctx, path, userID, data)
}

// Save saves changes to a file for the specified user at the specified path. It creates it if it
// does not exist
func Save(ctx context.Context, p string, userID int, data []byte) error {
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
			return saveFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID, data)
		}
	}
	return errors.New("No such root source")
}

func newFolderFromSource(ctx context.Context, source *fs.Source, path string, userID int) error {
	src, err := expandSource(source)
	if err != nil {
		return err
	}
	return src.NewFolder(ctx, path, userID)
}

// NewFolder creates a new folder.
func NewFolder(ctx context.Context, p string, userID int) error {
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
			return newFolderFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID)
		}
	}
	return errors.New("No such root source")
}
