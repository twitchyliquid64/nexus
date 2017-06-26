package integration

import (
	"context"
	"database/sql"
	"sync"
)

var db *sql.DB
var mapLock sync.Mutex
var runs map[string]*Run

type builtin interface {
	Apply(r *Run) error
}

// Initialise is called before all other methods to inject handles to dependencies.
func Initialise(ctx context.Context, database *sql.DB) error {
	db = database
	runs = map[string]*Run{}
	return nil
}
