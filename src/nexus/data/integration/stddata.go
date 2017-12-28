package integration

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/dlock"
	"nexus/data/util"
	"time"
)

// ErrNoStdRow symbolises that the row does not exist
var ErrNoStdRow = errors.New("Could not find row with that key")

// StdDataTable implements the databaseTable interface.
type StdDataTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *StdDataTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS integration_stddata (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    integration_parent INT NOT NULL,
	  modified_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    key varchar(1024) NOT NULL,
    value TEXT NOT NULL
	);

  CREATE INDEX IF NOT EXISTS integration_stddata_combined ON integration_stddata(integration_parent, key);
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
func (t *StdDataTable) Forms() []*util.FormDescriptor {
	return nil
}

// StdDataRow is a DAO for a data object in simple kv storage.
type StdDataRow struct {
	UID        int
	ParentUID  int
	ModifiedAt time.Time
	Key, Value string
}

// GetStdData returns a value for a given key
func GetStdData(ctx context.Context, runUID int, key string, db *sql.DB) (*StdDataRow, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT rowid, integration_parent, modified_at, key, value FROM integration_stddata WHERE integration_parent = ? AND key = ?;
	`, runUID, key)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrNoStdRow
	}

	var o StdDataRow
	return &o, res.Scan(&o.UID, &o.ParentUID, &o.ModifiedAt, &o.Key, &o.Value)
}

// WriteStdData saves datastore information
func WriteStdData(ctx context.Context, runUID int, key, value string, db *sql.DB) error {
	o, errFetch := GetStdData(ctx, runUID, key, db)
	if errFetch != nil && errFetch != ErrNoStdRow {
		return errFetch
	}

	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if errFetch == ErrNoStdRow {
		_, err = tx.Exec(`
      INSERT INTO
        integration_stddata (integration_parent, key, value)
        VALUES (
          ?, ?, ?
        );
    `, runUID, key, value)
	} else {
		_, err = tx.Exec(`
      UPDATE
        integration_stddata SET value = ?, modified_at = CURRENT_TIMESTAMP
      WHERE rowid = ?;
    `, value, o.UID)
	}

	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
