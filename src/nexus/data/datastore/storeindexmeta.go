package datastore

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"time"
)

// IndexMetaTable (datastore_index_meta) implements the databaseTable interface.
type IndexMetaTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *IndexMetaTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS datastore_index_meta (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  datastore INT NOT NULL,
	  name varchar(128) NOT NULL,
    cols varchar(1024) NOT NULL,
		unique_index BOOLEAN NOT NULL DEFAULT FALSE,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

  CREATE INDEX IF NOT EXISTS datastore_index_meta_index_datastore ON datastore_index_meta(datastore);
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
func (t *IndexMetaTable) Forms() []*util.FormDescriptor {
	return nil
}

// Index represents the metadata associated with an index on the datastore.
type Index struct {
	UID       int
	Datastore int
	Name      string
	Cols      string
	CreatedAt time.Time
}

func makeIndex(ctx context.Context, tx *sql.Tx, ind *Index, db *sql.DB) (int, error) {
	x, err := tx.ExecContext(ctx, `
		INSERT INTO
			datastore_index_meta (datastore, name, cols)
			VALUES (
				?, ?, ?
			);`, ind.Datastore, ind.Name, ind.Cols)
	id, err := x.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), err
}

// GetIndexes gets all the indexes for a datastore.
func GetIndexes(ctx context.Context, datastoreID int, db *sql.DB) ([]*Index, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, name, cols, created_at FROM datastore_index_meta WHERE datastore=?;`, datastoreID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Index
	for res.Next() {
		var out Index
		if err := res.Scan(&out.UID, &out.Name, &out.Cols, &out.CreatedAt); err != nil {
			return nil, err
		}
		out.Datastore = datastoreID
		output = append(output, &out)
	}
	return output, nil
}
