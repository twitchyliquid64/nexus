package fs

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"os"
	"time"
)

// source types
const (
	FSSourceLocal int = iota
	FSSourceMiniFS
	FSSourceS3
)

// SourceTable (fs_sources) implements the databaseTable interface.
type SourceTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *SourceTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS fs_sources (
    owner_uid INT NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
    prefix STRING NOT NULL,
    kind INT NOT NULL,

    value1 STRING NOT NULL,
    value2 STRING NOT NULL,
    value3 STRING NOT NULL,
	);

  CREATE INDEX IF NOT EXISTS fs_sources_by_owner ON fs_sources(owner_uid);
  CREATE INDEX IF NOT EXISTS fs_sources_by_prefix ON fs_sources(prefix);
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
func (t *SourceTable) Forms() []*util.FormDescriptor {
	return []*util.FormDescriptor{
		&util.FormDescriptor{
			FormTitle: "Filesystem Sources",
			ID:        "fsUserSources",
		},
	}
}

// Source represents a filesystem source for a user
type Source struct {
	UID     int
	OwnerID int
	Prefix  string
	Kind    int

	CreatedAt time.Time

	Value1, Value2, Value3 string
}

// GetSource returns a source with the given prefix and owner.
func GetSource(ctx context.Context, ownerUID int, prefix string, db *sql.DB) (*Source, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), owner_uid, created_at, prefix, kind, value1, value2, value3 FROM fs_sources WHERE owner_uid= $1 AND prefix = $2;
	`, ownerUID, prefix)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, os.ErrNotExist
	}

	var o Source
	return &o, res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Prefix, &o.Kind, &o.Value1, &o.Value2, &o.Value3)
}

// GetSourcesForUser returns sources for a given user.
func GetSourcesForUser(ctx context.Context, ownerUID int, db *sql.DB) ([]*Source, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), owner_uid, created_at, prefix, kind, value1, value2, value3 FROM fs_sources WHERE owner_uid= $1;
	`, ownerUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Source
	for res.Next() {
		var o Source
		if err := res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Prefix, &o.Kind, &o.Value1, &o.Value2, &o.Value3); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// CreateSource creates a source.
func CreateSource(ctx context.Context, source *Source, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	e, err := tx.Exec(`
      INSERT INTO
        fs_sources (owner_uid, prefix, kind, value1, value2, value3)
        VALUES (
          $1, $2,	$3, $4, $5, $6
        );
    `, source.OwnerID, source.Prefix, source.Kind, source.Value1, source.Value2, source.Value3)
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
