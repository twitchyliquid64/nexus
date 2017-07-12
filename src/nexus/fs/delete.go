package fs

import (
	"context"
	"errors"
	"nexus/data/fs"
	"strings"
)

func deleteMiniFS(ctx context.Context, p string, userID int) error {
	return errors.New("Not implemented")
}

func deleteFromSource(ctx context.Context, source *fs.Source, p string, userID int) error {
	return errors.New("Not implemented")
}

// Delete removes a file from the filesystem, throwing an error if it doesnt exist.
func Delete(ctx context.Context, p string, userID int) error {
	if err := validatePath(p); err != nil {
		return err
	}

	if strings.HasPrefix(p, "/minifs") {
		return deleteMiniFS(ctx, p[len("/minifs"):], userID)
	}

	// identify the source and query that
	sources, err := fs.GetSourcesForUser(ctx, userID, db)
	if err != nil {
		return err
	}

	splitPath := strings.Split(p, "/")
	if len(splitPath) <= 2 {
		return errors.New("Expected at least two path components")
	}
	for _, source := range sources {
		if splitPath[1] == source.Prefix {
			return deleteFromSource(ctx, source, strings.Join(splitPath, "/"), userID)
		}
	}
	return errors.New("No such root source")
}
