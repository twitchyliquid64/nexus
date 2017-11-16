package fs

import (
	"context"
	"io"
	"nexus/data/fs"
	"os"
	"strings"
)

func contentsFromSource(ctx context.Context, source *fs.Source, path string, userID int, writer io.Writer) error {
	src, err := ExpandSource(source)
	if err != nil {
		return err
	}
	return src.Contents(ctx, path, userID, writer)
}

// Contents writes the contents of the specified file to the given io.Writer.
func Contents(ctx context.Context, path string, userID int, writer io.Writer) error {
	var err error
	if path, err = validatePath(path); err != nil {
		return err
	}

	// identify the source and query that
	sources, err := getSourcesForUser(ctx, userID)
	if err != nil {
		return err
	}

	splitPath := strings.Split(path, "/")
	for _, source := range sources {
		if splitPath[1] == source.Prefix {
			return contentsFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID, writer)
		}
	}
	return os.ErrNotExist
}
