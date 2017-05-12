package user

import (
	"context"
	"database/sql"
)

// Table (user) implements the databaseTable interface.
type Table struct{}

// Called to create necessary structures in the database.
func (t *Table) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS users (
	  username STRING NOT NULL,
	  display_name STRING,
	  created_at TIMESTAMP NOT NULL,
	  passhash_if_no_auth_methods STRING
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
