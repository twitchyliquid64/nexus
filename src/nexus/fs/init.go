package fs

import (
	"context"
	"database/sql"
)

// fs implements a middleware layer for accessing a users virtual filesystem.
var db *sql.DB

// Initialize is called to start the database system.
func Initialize(ctx context.Context, database *sql.DB) error {
	db = database
	return nil
}
