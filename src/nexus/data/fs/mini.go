package fs

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"nexus/data/dlock"
	"nexus/data/util"
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
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_uid INT NOT NULL,
	  modified_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    path varchar(512) NOT NULL,
    access_level INT NOT NULL DEFAULT 0,
    kind INT NOT NULL DEFAULT 0,
    data BLOB NOT NULL
	);

  CREATE INDEX IF NOT EXISTS fs_minifiles_combined ON fs_minifiles(path, owner_uid);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *MiniFsTable) Forms() []*util.FormDescriptor {
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
    SELECT data FROM fs_minifiles WHERE rowid = ?;
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
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, modified_at, path, access_level, kind FROM fs_minifiles WHERE path = ? AND owner_uid= ?;
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
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := tx.QueryContext(ctx, `
    SELECT rowid FROM fs_minifiles WHERE path = ? AND owner_uid = ?;
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

// MiniFSDeleteFile deletes a file.
func MiniFSDeleteFile(ctx context.Context, f *File, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	r, err := tx.Exec(`
    DELETE FROM
      fs_minifiles
    WHERE owner_uid = ? AND path = ?;
  `, f.OwnerID, f.Path)
	if affected, errAffected := r.RowsAffected(); affected == 0 || errAffected != nil {
		if errAffected != nil {
			err = errAffected
		} else {
			err = os.ErrNotExist
		}
	}
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
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

	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	if f.UID != 0 { //either we found it or we already knew the UID
		_, errUpdate := tx.Exec(`
      UPDATE
        fs_minifiles
      SET
        access_level=?, kind=?, data=?, modified_at=CURRENT_TIMESTAMP
      WHERE rowid = ?;
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
          ?, ?, ?, ?, ?
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
