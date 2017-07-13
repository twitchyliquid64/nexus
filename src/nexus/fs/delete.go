package fs

import (
	"context"
	"errors"
	"io/ioutil"
	"nexus/data/fs"
	"os"
	"path"
	"strings"
)

// ErrHasFiles is returned if one attempts to delete a non-empty directory.
var ErrHasFiles = errors.New("Cannot delete non-empty directory")

func deleteMiniFS(ctx context.Context, p string, userID int) error {
	f, err := fs.MiniFSGetFile(ctx, userID, p, db)
	if err != nil {
		return err
	}
	if f.Kind == fs.FSKindDirectory {
		r, err2 := f.GetReader(ctx, db)
		if err2 != nil {
			return err2
		}
		d, err2 := ioutil.ReadAll(r)
		if err2 != nil {
			return err2
		}
		if len(d) > 0 {
			return ErrHasFiles
		}
	}

	err = deleteMiniFSFromDirectory(ctx, p, userID)
	if err != nil {
		return err
	}
	return fs.MiniFSDeleteFile(ctx, &fs.File{
		OwnerID: userID,
		Path:    p,
	}, db)
}

func deleteFromSource(ctx context.Context, source *fs.Source, p string, userID int) error {
	return errors.New("Not implemented")
}

func deleteMiniFSFromDirectory(ctx context.Context, p string, userID int) error {
	// check/update the directory file
	dir, err := fs.MiniFSGetFile(ctx, userID, path.Dir(p), db)
	if err == os.ErrNotExist {
		return nil
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

	if strings.Contains(string(listingData), "\n"+p+"\n") {
		dir.CachedData = []byte(strings.Replace(string(listingData), "\n"+p+"\n", "", -1))
		_, err = fs.MiniFSSaveFile(ctx, dir, db)
	}
	return err
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
