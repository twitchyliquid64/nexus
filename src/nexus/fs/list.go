package fs

import (
	"bufio"
	"context"
	"errors"
	"log"
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

func listMiniFSFiles(ctx context.Context, path string, userID int) ([]ListResultItem, error) {
	if path == "" {
		path = "/"
	}

	f, err := fs.MiniFSGetFile(ctx, userID, path, db)
	if err == os.ErrNotExist && path == "/" {
		id, errDir := fs.MiniFSSaveFile(ctx, &fs.File{
			OwnerID:     userID,
			Kind:        fs.FSKindDirectory,
			AccessLevel: fs.FSAccessPrivate,
			Path:        "/",
		}, db)
		log.Printf("[FS] Made root directory for miniFS - UID: %d, Err: %v", id, errDir)
		return nil, errDir
	}
	if err != nil {
		return nil, err
	}

	if f.Kind != fs.FSKindDirectory {
		return nil, errors.New("Specified path is not a directory")
	}
	listing, err := f.GetReader(ctx, db)
	if err != nil {
		return nil, err
	}

	var output []ListResultItem
	iterator := bufio.NewScanner(listing)
	for iterator.Scan() {
		if iterator.Text() == "" {
			continue
		}
		fileInfo, err := fs.MiniFSGetFile(ctx, userID, iterator.Text(), db)
		if err != nil {
			return nil, err
		}

		output = append(output, ListResultItem{
			Name:     fileInfo.Path,
			Modified: fileInfo.ModifiedAt,
			ItemKind: miniFSKindToItemKind(fileInfo.Kind),
		})
	}
	if err := iterator.Err(); err != nil {
		return nil, err
	}
	return output, nil
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
	return nil, errors.New("Not implemented")
}

// List returns a list of files at the specified path for the specified user.
func List(ctx context.Context, path string, userID int) ([]ListResultItem, error) {
	if err := validatePath(path); err != nil {
		return nil, err
	}

	if path == "/" { //special case - list root sources
		return listSources(ctx, path, userID)
	}

	if strings.HasPrefix(path, "/minifs") {
		return listMiniFSFiles(ctx, path[len("/minifs"):], userID)
	}

	// identify the source and query that
	sources, err := fs.GetSourcesForUser(ctx, userID, db)
	if err != nil {
		return nil, err
	}

	splitPath := strings.Split(path, "/")
	if len(splitPath) <= 2 {
		return nil, errors.New("Expected at least two path components")
	}
	for _, source := range sources {
		if splitPath[1] == source.Prefix {
			return listFromSource(ctx, source, strings.Join(splitPath, "/"), userID)
		}
	}
	return nil, os.ErrNotExist
}
