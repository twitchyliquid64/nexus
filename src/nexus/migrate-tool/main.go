package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/cznic/ql"
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

func dumpTable(table string, db *ql.DB, w io.Writer) error {
	tableInfo, err := tableInfo(table, db)
	if err != nil {
		return err
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

	w.Write([]byte(";\n\n"))
	return nil
}

func main() {
	ql.RegisterDriver()
	flag.Parse()
	if flag.Arg(0) == "" {
		fmt.Fprintf(os.Stderr, "Error: Must supply a ql database file to read from.\n")
		os.Exit(1)
	}

	qlDb, err := ql.OpenFile(flag.Arg(0), &ql.Options{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Err opening database: %s\n", err)
		os.Exit(2)
	}

	for _, table := range tablesToMigrate {
		err := dumpTable(table, qlDb, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed dumping %q: %s\n", table, err)
			os.Exit(3)
		}
	}
}
