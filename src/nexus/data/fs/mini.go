package fs

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"os"
	"time"
)

// entry types
const (
	FSKindFile int = iota
	FSKindDirectory
)

// access levels
const (
	FSAccessPrivate int = iota
	FSAccessAuthenticated
	FSAccessPublic
)

// MiniFsTable (miniFS) implements the databaseTable interface.
type MiniFsTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *MiniFsTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS fs_minifiles (
    owner_uid INT NOT NULL,
	  modified_at TIME NOT NULL DEFAULT now(),
    path STRING NOT NULL,
    access_level INT NOT NULL DEFAULT 0,
    kind INT NOT NULL DEFAULT 0,
    data BLOB NOT NULL,
	);

  CREATE INDEX IF NOT EXISTS fs_minifiles_by_owner ON fs_minifiles(owner_uid);
  CREATE INDEX IF NOT EXISTS fs_minifiles_by_path ON fs_minifiles(path);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// File represents a file or directory stored in miniFS
// it will implement whatever interface I use for files in the fs subsystem.
type File struct {
	UID     int
	OwnerID int
	Path    string

	ModifiedAt time.Time

	AccessLevel int
	Kind        int

	CachedData []byte
}

// GetReader returns a reader which can be used to get file information.
func (f *File) GetReader(ctx context.Context, db *sql.DB) (io.Reader, error) {
	res, err := db.QueryContext(ctx, `
    SELECT data FROM fs_minifiles WHERE id() = $1;
  `, f.UID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, os.ErrNotExist
	}

	var o []byte
	err = res.Scan(&o)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(o), nil
}

// MiniFSGetFile by a user's UID and a file path.
func MiniFSGetFile(ctx context.Context, ownerUID int, path string, db *sql.DB) (*File, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), owner_uid, modified_at, path, access_level, kind FROM fs_minifiles WHERE path = $1 AND owner_uid= $2;
	`, path, ownerUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, os.ErrNotExist
	}

	var o File
	return &o, res.Scan(&o.UID, &o.OwnerID, &o.ModifiedAt, &o.Path, &o.AccessLevel, &o.Kind)
}

// MiniFSFileExists returns true if the given file exists.
func MiniFSFileExists(ctx context.Context, tx *sql.Tx, path string, ownerUID int, db *sql.DB) (bool, int, error) {
	res, err := tx.QueryContext(ctx, `
    SELECT id() FROM fs_minifiles WHERE path = $1 AND owner_uid = $2;
  `, path, ownerUID)
	if err != nil {
		return false, 0, err
	}
	defer res.Close()

	if !res.Next() {
		return false, 0, nil
	}
	var o int
	return true, o, res.Scan(&o)
}

// MiniFSSaveFile saves a file in miniFS. DO NOT use for ownership transfers or renames.
func MiniFSSaveFile(ctx context.Context, f *File, db *sql.DB) (int, error) {
	var err error
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	if f.UID == 0 { //UID not already known - so file could possible exist
		_, f.UID, err = MiniFSFileExists(ctx, tx, f.Path, f.OwnerID, db)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	if f.UID != 0 { //either we found it or we already knew the UID
		_, errUpdate := tx.Exec(`
      UPDATE
        fs_minifiles
      SET
        access_level=$1, kind=$2, data=$3
      WHERE id() = $4;
    `, f.AccessLevel, f.Kind, f.CachedData, f.UID)
		if errUpdate != nil {
			tx.Rollback()
			return 0, err
		}
		return f.UID, tx.Commit()
	}

	//doesn't exist yet
	e, err := tx.Exec(`
      INSERT INTO
        fs_minifiles (owner_uid, path, access_level, kind, data)
        VALUES (
          $1, $2,	$3, $4, $5
        );
    `, f.OwnerID, f.Path, f.AccessLevel, f.Kind, f.CachedData)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	lastID, err := e.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return int(lastID), tx.Commit()
}
