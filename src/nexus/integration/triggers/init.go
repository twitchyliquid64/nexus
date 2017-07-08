package triggers

import "database/sql"

var db *sql.DB

// Initialise is called on startup to store a reference to the database handle.
func Initialise(dbIn *sql.DB) {
	db = dbIn
}
