package fs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    prefix varchar(128) NOT NULL,
    kind INT NOT NULL,

    value1 varchar(1024) NOT NULL,
    value2 varchar(1024) NOT NULL,
    value3 varchar(1024) NOT NULL
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
			SettingsTitle: "Filesystem Sources",
			ID:            "fsUserSources",
			Desc:          "Sources are additional mount points available in your filesystem. For instance, a 'source' can be a S3 bucket or folder in the server's local filesystem.",
			Forms: []*util.ActionDescriptor{
				&util.ActionDescriptor{
					Name:   "New Source",
					ID:     "fs_source_add",
					IcoStr: "create_new_folder",
					Fields: []*util.Field{
						&util.Field{
							Name: "Source Type",
							ID:   "kind",
							Kind: "select",
							SelectOptions: map[string]string{
								fmt.Sprintf("%d", FSSourceS3): "S3 Bucket",
							},
						},
						&util.Field{
							Name:              "Mount-point name",
							ID:                "prefix",
							Kind:              "text",
							ValidationPattern: "[A-Za-z_\\s]{1,18}",
						},
						&util.Field{
							Name: "Bucket Name & Region (bucketname:region)",
							ID:   "val1",
							Kind: "text",
						},
						&util.Field{
							Name: "Access Key",
							ID:   "val2",
							Kind: "text",
						},
						&util.Field{
							Name: "Secret Key",
							ID:   "val3",
							Kind: "text",
						},
					},
					OnSubmit: t.addSourceSubmitHandler,
				},
			},
			Tables: []*util.TableDescriptor{
				&util.TableDescriptor{
					Name: "Existing Sources",
					ID:   "fs_existing_sources",
					Cols: []string{"#", "Prefix", "Type", "Val1"},
					FetchContent: func(ctx context.Context, userID int, db *sql.DB) ([]interface{}, error) {
						data, err := GetSourcesForUser(ctx, userID, db)
						if err != nil {
							return nil, err
						}
						out := make([]interface{}, len(data))
						for i, s := range data {
							out[i] = []interface{}{s.UID, s.Prefix, s.Kind, s.Value1}
						}
						return out, nil
					},
				},
			},
		},
	}
}

// Called on the submission of the form 'New Source'
func (t *SourceTable) addSourceSubmitHandler(ctx context.Context, vals map[string]string, userID int, db *sql.DB) error {
	kind := FSSourceS3
	//TODO: When we support more sources do this properly
	if vals["kind"] != fmt.Sprintf("%d", FSSourceS3) {
		return errors.New("Invalid source type")
	}

	_, err := CreateSource(ctx, &Source{
		OwnerID: userID,
		Prefix:  vals["prefix"],
		Kind:    kind,
		Value1:  vals["val1"],
		Value2:  vals["val2"],
		Value3:  vals["val3"],
	}, db)
	return err
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
		SELECT rowid, owner_uid, created_at, prefix, kind, value1, value2, value3 FROM fs_sources WHERE owner_uid = ? AND prefix = ?;
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
		SELECT rowid, owner_uid, created_at, prefix, kind, value1, value2, value3 FROM fs_sources WHERE owner_uid = ?;
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
          ?, ?, ?, ?, ?, ?
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
