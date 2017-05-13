package datastore

import (
	"context"
	"database/sql"
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
	  owner_uid int NOT NULL,
	  name STRING NOT NULL,
		store_kind STRING NOT NULL DEFAULT "DB",
	  created_at TIME NOT NULL DEFAULT now(),
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

// Datastore represents a datastore in the system.
type Datastore struct {
	UID       int
	Name      string
	OwnerID   int
	Kind      string
	CreatedAt time.Time
}

// MakeDatastore registers a column.
func MakeDatastore(ctx context.Context, ds *Datastore, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	x, err := tx.ExecContext(ctx, `
		INSERT INTO
			datastore_meta (owner_uid, name, store_kind)
			VALUES (
				$1, $2, $3
			);`, ds.OwnerID, ds.Name, string(ds.Kind))
	if err != nil {
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), tx.Commit()
}

// GetDatastores gets all datastores owned by that user. If showAll is true, then all datastores are returned.
func GetDatastores(ctx context.Context, showAll bool, userID int, db *sql.DB) ([]*Datastore, error) {
	res, err := db.QueryContext(ctx, `SELECT id(), name, owner_uid, store_kind, created_at FROM datastore_meta WHERE owner_uid=$1 OR $2;`, userID, showAll)
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
