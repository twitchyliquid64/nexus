package fs

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"nexus/data/fs"
	"os"
	"path"
	"strings"
)

func saveMiniFS(ctx context.Context, p string, userID int, data []byte) error {
	err := saveMiniFSDirectory(ctx, p, userID)
	if err != nil {
		return err
	}

	_, err = fs.MiniFSSaveFile(ctx, &fs.File{
		Path:        p,
		CachedData:  data,
		OwnerID:     userID,
		Kind:        fs.FSKindFile,
		AccessLevel: fs.FSAccessPrivate,
	}, db)
	return err
}

func saveMiniFSDirectory(ctx context.Context, p string, userID int) error {
	// check/update the directory file
	dir, err := fs.MiniFSGetFile(ctx, userID, path.Dir(p), db)
	if err == os.ErrNotExist {
		id, errNew := fs.MiniFSSaveFile(ctx, &fs.File{
			OwnerID:     userID,
			Kind:        fs.FSKindDirectory,
			AccessLevel: fs.FSAccessPrivate,
			Path:        "\n" + p + "\n",
		}, db)
		log.Printf("[FS] Made %s directory for miniFS - UID: %d, Err: %v", path.Dir(p), id, errNew)
		return errNew
	} else if err != nil {
		return err
	}

	if dir.Kind != fs.FSKindDirectory {
		return errors.New("Cannot base a file off another file")
	}
	listing, err := dir.GetReader(ctx, db)
	if err != nil {
		return err
	}
	listingData, err := ioutil.ReadAll(listing)
	if err != nil {
		return err
	}

	if !strings.Contains(string(listingData), "\n"+p+"\n") { //doesnt exist, add it to the directory
		dir.CachedData = []byte(string(listingData) + "\n" + p + "\n")
		_, err := fs.MiniFSSaveFile(ctx, dir, db)
		if err != nil {
			return err
		}
	}
	return nil
}

func saveFromSource(ctx context.Context, source *fs.Source, path string, userID int, data []byte) error {
	return errors.New("Not implemented")
}

// Save saves changes to a file for the specified user at the specified path. It creates it if it
// does not exist
func Save(ctx context.Context, p string, userID int, data []byte) error {
	if err := validatePath(p); err != nil {
		return err
	}

	if strings.HasPrefix(p, "/minifs") {
		return saveMiniFS(ctx, p[len("/minifs"):], userID, data)
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
			return saveFromSource(ctx, source, strings.Join(splitPath, "/"), userID, data)
		}
	}
	return errors.New("No such root source")
}

func newFolderMiniFS(ctx context.Context, p string, userID int) error {
	err := saveMiniFSDirectory(ctx, p, userID)
	if err != nil {
		return err
	}

	_, err = fs.MiniFSSaveFile(ctx, &fs.File{
		Path:        p,
		OwnerID:     userID,
		Kind:        fs.FSKindDirectory,
		AccessLevel: fs.FSAccessPrivate,
	}, db)
	return err
}

func newFolderFromSource(ctx context.Context, source *fs.Source, path string, userID int) error {
	return errors.New("Not implemented")
}

// NewFolder creates a new folder.
func NewFolder(ctx context.Context, p string, userID int) error {
	if err := validatePath(p); err != nil {
		return err
	}

	if strings.HasPrefix(p, "/minifs") {
		return newFolderMiniFS(ctx, p[len("/minifs"):], userID)
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
			return newFolderFromSource(ctx, source, strings.Join(splitPath, "/"), userID)
		}
	}
	return errors.New("No such root source")
}
