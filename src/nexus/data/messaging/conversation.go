package messaging

import (
	"context"
	"database/sql"
)

// ConversationTable implements the DataTable interface.
type ConversationTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *ConversationTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS messaging_conversation (
	  name STRING NOT NULL,
    source_uid INT NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
	  source_unique_identifier STRING NOT NULL,
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
