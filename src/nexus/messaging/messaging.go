package messaging

import (
	"context"
	"database/sql"
)

// Init starts the local messaging system - fetching and delivering messages for all non-remote sources.
func Init(ctx context.Context, db *sql.DB) error {
	return nil
}
