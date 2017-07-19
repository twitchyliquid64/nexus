package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"nexus/data"
	"os"
	"reflect"
	"strings"
	"time"

	"database/sql"

	"github.com/cznic/ql"
	// load sqlite library
	_ "github.com/mattn/go-sqlite3"
)

var tablesToMigrate = []string{
	"fs_sources",
	"fs_minifiles",
	"users",
	"user_auth",
	"sessions",
	"messaging_source",
	"messaging_messages",
	"messaging_conversation",
	"integration_trigger",
	"integration_stddata",
	"integration_runnable",
	"integration_log",
}

func tableInfo(table string, db *ql.DB) (*ql.TableInfo, error) {
	dbInfo, err := db.Info()
	if err != nil {
		return nil, err
	}
	for _, t := range dbInfo.Tables {
		if t.Name == table {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("no such table %q", table)
}

func getNumRows(table string, db *ql.DB) (int, error) {
	res, _, err := db.Run(ql.NewRWCtx(), "SELECT count(*) FROM "+table+";")
	if err != nil {
		return 0, err
	}
	var amount = 0
	for _, r := range res {
		err := r.Do(false, func(data []interface{}) (bool, error) {
			amount = int(data[0].(int64))
			return false, nil
		})
		if err != nil {
			return 0, err
		}
	}
	return amount, nil
}

func dumpTable(table string, db *ql.DB, w io.Writer) error {
	tableInfo, err := tableInfo(table, db)
	if err != nil {
		return err
	}

	n, err := getNumRows(table, db)
	if err != nil {
		return err
	}
	if n == 0 {
		return nil
	}

	queryStr := "id(), "
	w.Write([]byte(fmt.Sprintf("INSERT INTO %s (rowid, ", table)))
	for i, c := range tableInfo.Columns {
		w.Write([]byte(c.Name))
		queryStr += c.Name
		if i < (len(tableInfo.Columns) - 1) {
			w.Write([]byte(", "))
			queryStr += ", "
		}
	}
	w.Write([]byte(") VALUES \n"))

	queryStr = "SELECT " + queryStr + " FROM " + table + ";"
	res, _, err := db.Run(ql.NewRWCtx(), queryStr)
	if err != nil {
		return err
	}
	shouldWriteComma := false
	for _, r := range res {
		err := r.Do(false, func(data []interface{}) (bool, error) {
			if len(data) > 0 {
				if shouldWriteComma {
					w.Write([]byte(",\n"))
				}
				shouldWriteComma = true
				w.Write([]byte("("))

				for i, v := range data {

					switch val := v.(type) {
					case bool:
						if val {
							w.Write([]byte("1"))
						} else {
							w.Write([]byte("0"))
						}
					case []uint8:
						w.Write([]byte("X'" + hex.EncodeToString(val) + "'"))
					case nil:
						w.Write([]byte("NULL"))
					case int, int64:
						w.Write([]byte(fmt.Sprint(val)))
					case time.Time:
						w.Write([]byte("'" + val.Format(time.RFC3339) + "'"))
					case string:
						escapedString := strings.Replace(val, "'", "''", -1)
						w.Write([]byte("'" + escapedString + "'"))
					default:
						return true, fmt.Errorf("cannot handle type %q", reflect.TypeOf(v).String())
					}

					if i < (len(data) - 1) {
						w.Write([]byte(", "))
					}
				}
				w.Write([]byte(")"))

			}
			return true, nil
		})
		if err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte(";")); err != nil {
		return err
	}
	w.Write([]byte("\n\n"))
	return nil
}

func doCreateBlankDB(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err == nil {
		err := os.Remove(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed delete for old sqlite database: %s\n", err)
			os.Exit(6)
		}
	}

	return data.Init(context.Background(), "sqlite3", path)
}

type dbBatchExecutor struct {
	db   *sql.DB
	buff *bytes.Buffer
}

func (e *dbBatchExecutor) Write(p []byte) (int, error) {
	n, err := e.buff.Write(p)
	if err != nil {
		return n, err
	}
	if strings.HasSuffix(string(p), ";") {
		r, err := e.db.Exec(string(e.buff.String()))
		if err != nil {
			return 0, err
		}
		a, _ := r.LastInsertId()
		n, _ := r.RowsAffected()
		fmt.Printf("[%d,%d]: %q\n", n, a, e.buff.String())
		e.buff.Reset()
	}
	return len(p), nil
}

func wrapDbToWriter(db *sql.DB) io.Writer {
	b := bytes.NewBuffer(make([]byte, 0, 1024*8))
	return &dbBatchExecutor{
		db:   db,
		buff: b,
	}
}

func main() {
	ql.RegisterDriver()
	var newNamePath string
	flag.StringVar(&newNamePath, "new_db", "", "Optional path to new db to initialise. Will dump directly if specified.")
	flag.Parse()

	writer := io.Writer(os.Stdout)
	if newNamePath != "" {
		db, err := doCreateBlankDB(newNamePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing sqlite DB: %v\n", err)
			os.Exit(6)
		}
		defer db.Close()
		writer = wrapDbToWriter(db)
	}

	if flag.Arg(0) == "" {
		fmt.Fprintf(os.Stderr, "Error: Must supply a ql database file to read from.\n")
		os.Exit(1)
	}

	qlDb, err := ql.OpenFile(flag.Arg(0), &ql.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Err opening ql database: %s\n", err)
		os.Exit(2)
	}

	for _, table := range tablesToMigrate {
		err := dumpTable(table, qlDb, writer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed dumping %q: %s\n", table, err)
			os.Exit(3)
		}
	}
}
