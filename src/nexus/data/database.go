package data

import (
	"context"
	"database/sql"
	"log"
	"nexus/data/datastore"
	"nexus/data/fs"
	"nexus/data/integration"
	"nexus/data/mc"
	"nexus/data/messaging"
	"nexus/data/session"
	"nexus/data/user"
	"nexus/data/util"
	"reflect"

	sqlite3 "github.com/mattn/go-sqlite3"
)

var tables = []DatabaseTable{
	&user.Table{},
	&user.AuthTable{},
	&user.AttrTable{},
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

	&mc.APIKeyTable{},
	&mc.StatusTable{},
	&mc.LocationTable{},
}

var sqlite3conn = []*sqlite3.SQLiteConn{}
var sqlite3backupconn = []*sqlite3.SQLiteConn{}

// GetSQLiteConn returns the underlying database connection.
func GetSQLiteConn() *sqlite3.SQLiteConn {
	return sqlite3conn[0]
}

// DatabaseTable represents the manager object for a database table.
type DatabaseTable interface {
	Setup(ctx context.Context, db *sql.DB) error
	Forms() []*util.FormDescriptor
}

// Init is called with database information to initialise a database session, creating any necessary tables.
func Init(ctx context.Context, connString string) (*sql.DB, error) {
	sql.Register("sqlite3_conn_hook_main",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				sqlite3conn = append(sqlite3conn, conn)
				return nil
			},
		})
	sql.Register("sqlite3_conn_hook_backup",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				sqlite3backupconn = append(sqlite3backupconn, conn)
				return nil
			},
		})

	db, err := sql.Open("sqlite3_conn_hook_main", connString)
	if err != nil {
		return nil, err
	}

	for _, table := range tables {
		err := table.Setup(ctx, db)
		if err != nil {
			log.Printf("Problem initialising: %s", reflect.TypeOf(table))
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

// GetForms returns all the forms which are registered by database tables.
func GetForms() []*util.FormDescriptor {
	var forms []*util.FormDescriptor

	for _, table := range tables {
		tableForms := table.Forms()
		if tableForms != nil {
			forms = append(forms, tableForms...)
		}
	}
	return forms
}

// Vacuum is called to compress the database.
func Vacuum(db *sql.DB) error {
	_, err := db.Exec("VACUUM;")
	return err
}

type TableStat struct {
	Count int
}

// GetTableStats is called to return statistics about each table.
func GetTableStats(ctx context.Context, db *sql.DB) (map[string]TableStat, error) {
	tables, err := db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table';")
	if err != nil {
		return nil, err
	}

	var tableNames []string
	for tables.Next() {
		var name string
		err2 := tables.Scan(&name)
		if err2 != nil {
			return nil, err2
		}
		tableNames = append(tableNames, name)
	}
	countsByTable := map[string]TableStat{}

	for _, tableName := range tableNames {
		row := db.QueryRowContext(ctx, "SELECT COUNT() FROM "+tableName+";")
		var num int
		err2 := row.Scan(&num)
		if err2 != nil {
			return nil, err2
		}
		countsByTable[tableName] = TableStat{num}
	}

	return countsByTable, nil
}
