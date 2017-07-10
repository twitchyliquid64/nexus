package data

import (
	"context"
	"database/sql"
	"nexus/data/datastore"
	"nexus/data/fs"
	"nexus/data/integration"
	"nexus/data/messaging"
	"nexus/data/session"
	"nexus/data/user"
	"reflect"

	"github.com/cznic/ql"
)

var tables = []DatabaseTable{
	&user.Table{},
	&user.AuthTable{},
	&session.Table{},
	&datastore.MetaTable{},
	&datastore.ColumnMetaTable{},
	&datastore.StoreGrant{},
	&messaging.SourceTable{},
	&messaging.ConversationTable{},
	&messaging.MessageTable{},
	&integration.Table{},
	&integration.TriggerTable{},
	&integration.LogTable{},
	&integration.StdDataTable{},
	&fs.MiniFsTable{},
	&fs.SourceTable{},
}

// DatabaseTable represents the manager object for a database table.
type DatabaseTable interface {
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

// GetTable returns the table manager for a given databaseTable type.
func GetTable(tbl DatabaseTable) DatabaseTable {
	for _, table := range tables {
		if reflect.TypeOf(table) == reflect.TypeOf(tbl) {
			return table
		}
	}
	return nil
}
