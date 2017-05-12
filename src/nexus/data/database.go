package data

import (
	"context"
	"database/sql"
	"nexus/data/user"

	"github.com/cznic/ql"
)

var tables = []databaseTable{
	&user.Table{},
}

type databaseTable interface {
	Setup(ctx context.Context, db *sql.DB) error
}

// Init is called with database information to initialise a database session, creating any necessary tables.
func Init(ctx context.Context, databaseKind, connString string) (*sql.DB, error) {
	ql.RegisterDriver()
	db, err := sql.Open(databaseKind, connString)
	if err != nil {
		return nil, err
	}

	for _, table := range tables {
		err := table.Setup(ctx, db)
		if err != nil {
			db.Close()
			return nil, err
		}
	}

	return db, nil
}
