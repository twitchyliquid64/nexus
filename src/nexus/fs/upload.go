package fs

import (
	"context"
	"errors"
	"io"
	"nexus/data/fs"
	"strings"
)

func uploadFromSource(ctx context.Context, source *fs.Source, path string, userID int, data io.Reader) error {
	src, err := ExpandSource(source)
	if err != nil {
		return err
	}
	return src.Upload(ctx, path, userID, data)
}

// Upload does a streaming save.
func Upload(ctx context.Context, p string, userID int, data io.Reader) error {
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
			return uploadFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID, data)
		}
	}
	return errors.New("No such root source")
}
