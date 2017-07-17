package datastore

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"time"
)

// ColumnMetaTable (storemeta) implements the databaseTable interface.
type ColumnMetaTable struct{}

// Datatype represents the kind of information stored in a column.
type Datatype int

// Available column datatypes
const (
	INT Datatype = iota
	UINT
	FLOAT
	STR
	BLOB
	TIME
)

// Setup is called on initialization to create necessary structures in the database.
func (t *ColumnMetaTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS datastore_col_meta (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  datastore INT NOT NULL,
	  name varchar(128) NOT NULL,
    datatype INT NOT NULL,
    ordering INT NOT NULL DEFAULT 0,
	  created_at TIMESTAMPSTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

  CREATE INDEX IF NOT EXISTS datastore_col_meta_index_datastore ON datastore_col_meta(datastore);
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
func (t *ColumnMetaTable) Forms() []*util.FormDescriptor {
	return nil
}

// Column represents the metadata associated with a datastore column.
type Column struct {
	UID       int
	Name      string
	Datatype  Datatype
	Ordering  int
	CreatedAt time.Time
}

// makeColumn registers a column.
func makeColumn(ctx context.Context, tx *sql.Tx, datastoreID int, col *Column, db *sql.DB) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO
			datastore_col_meta (datastore, name, datatype, ordering)
			VALUES (
				?, ?, ?, ?
			);`, datastoreID, col.Name, int(col.Datatype), col.Ordering)
	return err
}

// GetColumns gets all the columns for a datastore.
func GetColumns(ctx context.Context, datastoreID int, db *sql.DB) ([]*Column, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, name, datatype, ordering, created_at FROM datastore_col_meta WHERE datastore=? ORDER BY ordering ASC;`, datastoreID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Column
	for res.Next() {
		var out Column
		if err := res.Scan(&out.UID, &out.Name, &out.Datatype, &out.Ordering, &out.CreatedAt); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}

// ColDatatype returns a string representation of the Datatype.
func ColDatatype(dt Datatype) string {
	switch dt {
	case INT:
		return "int"
	case UINT:
		return "uint"
	case STR:
		return "text"
	case FLOAT:
		return "float"
	case BLOB:
		return "blob"
	case TIME:
		return "timestamp"
	}
	return "?UNK?"
}
