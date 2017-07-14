package fs

import (
	"bufio"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"nexus/data/fs"
	"os"
	"strings"
)

type miniFS struct{}

func (_ *miniFS) Contents(ctx context.Context, p string, userID int, writer io.Writer) error {
	f, err := fs.MiniFSGetFile(ctx, userID, p, db)
	if err != nil {
		return err
	}

	r, err := f.GetReader(ctx, db)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, r)
	return err
}

func (_ *miniFS) Save(ctx context.Context, p string, userID int, data []byte) error {
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
	dir, err := fs.MiniFSGetFile(ctx, userID, computeDir(p), db)
	if err == os.ErrNotExist {
		dir = &fs.File{
			OwnerID:     userID,
			Kind:        fs.FSKindDirectory,
			AccessLevel: fs.FSAccessPrivate,
			Path:        computeDir(p),
		}
		id, errNew := fs.MiniFSSaveFile(ctx, dir, db)
		log.Printf("[FS] Made %s directory for miniFS - UID: %d, Err: %v", computeDir(p), id, errNew)
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

func (_ *miniFS) Delete(ctx context.Context, p string, userID int) error {
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

func deleteMiniFSFromDirectory(ctx context.Context, p string, userID int) error {
	// check/update the directory file
	dir, err := fs.MiniFSGetFile(ctx, userID, computeDir(p), db)
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

func (_ *miniFS) List(ctx context.Context, path string, userID int) ([]ListResultItem, error) {
	f, err := fs.MiniFSGetFile(ctx, userID, path, db)
	if err == os.ErrNotExist && path == "" {
		id, errDir := fs.MiniFSSaveFile(ctx, &fs.File{
			OwnerID:     userID,
			Kind:        fs.FSKindDirectory,
			AccessLevel: fs.FSAccessPrivate,
			Path:        "",
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

func (_ *miniFS) NewFolder(ctx context.Context, p string, userID int) error {
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
