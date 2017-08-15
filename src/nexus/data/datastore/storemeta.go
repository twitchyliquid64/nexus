package datastore

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/util"
	"time"
)

// MetaTable (storemeta) implements the databaseTable interface.
type MetaTable struct{}

// StoreKind represents what kind of datastore it is.
type StoreKind string

// Kinds of datastores
const (
	KindDB StoreKind = "DB"
)

// Setup is called on initialization to create necessary structures in the database.
func (t *MetaTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS datastore_meta (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  owner_uid int NOT NULL,
	  name varchar(128) NOT NULL,
		store_kind varchar(16) NOT NULL DEFAULT "DB",
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
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
func (t *MetaTable) Forms() []*util.FormDescriptor {
	return nil
}

// Datastore represents a datastore in the system.
type Datastore struct {
	UID       int
	Name      string
	OwnerID   int
	Kind      string
	CreatedAt time.Time
	Cols      []*Column //must be manually populated
}

// makeDatastore registers a column.
func makeDatastore(ctx context.Context, tx *sql.Tx, ds *Datastore, db *sql.DB) (int, error) {
	x, err := tx.ExecContext(ctx, `
		INSERT INTO
			datastore_meta (owner_uid, name, store_kind)
			VALUES (
				?, ?, ?
			);`, ds.OwnerID, ds.Name, string(ds.Kind))
	if err != nil {
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// GetDatastore gets a datastore by ID.
func GetDatastore(ctx context.Context, uid int, db *sql.DB) (*Datastore, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, name, owner_uid, store_kind, created_at FROM datastore_meta WHERE rowid=?;`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, errors.New("Datastore not found")
	}
	var out Datastore
	if err := res.Scan(&out.UID, &out.Name, &out.OwnerID, &out.Kind, &out.CreatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDatastoreByName gets a datastore by name.
func GetDatastoreByName(ctx context.Context, name string, db *sql.DB) (*Datastore, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, name, owner_uid, store_kind, created_at FROM datastore_meta WHERE name=?;`, name)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, errors.New("Datastore not found")
	}
	var out Datastore
	if err := res.Scan(&out.UID, &out.Name, &out.OwnerID, &out.Kind, &out.CreatedAt); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDatastores gets all datastores owned by that user. If showAll is true, then all datastores are returned.
func GetDatastores(ctx context.Context, showAll bool, userID int, db *sql.DB) ([]*Datastore, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, name, owner_uid, store_kind, created_at
	FROM datastore_meta
	WHERE
		owner_uid=? OR ?;`, userID, showAll, userID) //TODO: Fix grants
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Datastore
	for res.Next() {
		var out Datastore
		if err := res.Scan(&out.UID, &out.Name, &out.OwnerID, &out.Kind, &out.CreatedAt); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}
