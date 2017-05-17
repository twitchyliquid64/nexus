package datastore

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"strconv"
	"time"
)

// DoStreamingInsert takes input as a CSV and inserts it to a database.
// TODO: Refactor
func DoStreamingInsert(ctx context.Context, data io.Reader, dsID int, colIDs []int, db *sql.DB) error {
	insertCols := make([]*Column, len(colIDs))
	cols, err := GetColumns(ctx, dsID, db)
	if err != nil {
		return err
	}

	queryStr := "INSERT INTO ds_" + strconv.Itoa(dsID) + " ("
	for i := range colIDs {
		insertCols[i] = getColNameByUID(cols, strconv.Itoa(colIDs[i]))
		if insertCols[i] == nil {
			return errors.New("Invalid columnID")
		}
		queryStr += columnName(insertCols[i].Name)
		if i < len(colIDs)-1 {
			queryStr += ", "
		}
	}
	queryStr += ") VALUES ("
	for i := range colIDs {
		queryStr += "$" + strconv.Itoa(i+1)
		if i < len(colIDs)-1 {
			queryStr += ", "
		}
	}
	queryStr += ");"
	log.Printf("Streaming insert query: %s\n", queryStr)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	r := csv.NewReader(data)
	inContainers := make([]interface{}, len(colIDs))

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tx.Rollback()
			return err
		}

		log.Println(row)
		for i := range inContainers {
			if insertCols[i].Datatype == INT || insertCols[i].Datatype == UINT {
				v, _ := strconv.Atoi(row[i])
				inContainers[i] = v
			} else {
				inContainers[i] = row[i]
			}
		}

		_, err = tx.Exec(queryStr, inContainers...)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// DoStreamingQuery writes the result of a query in CSV form.
func DoStreamingQuery(ctx context.Context, response io.Writer, query Query, db *sql.DB) error {
	cols, err := GetColumns(ctx, query.UID, db)
	if err != nil {
		return err
	}

	queryString, queryParameters, err := makeFullQuery(cols, query)
	if err != nil {
		return err
	}

	w := csv.NewWriter(response)
	defer w.Flush()

	// write headers
	row := []string{"UID"}
	for _, col := range cols {
		row = append(row, col.Name)
	}
	if err = w.Write(row); err != nil {
		return err
	}

	// setup query
	res, err := db.QueryContext(ctx, queryString, queryParameters...)
	if err != nil {
		return err
	}
	defer res.Close()

	// mainloop query
	out := make([]string, len(cols)+1)
	pointers := buildResultsetScanContainers(cols)
	for res.Next() {
		if err := res.Scan(pointers...); err != nil {
			return err
		}
		for i, val := range pointers {
			switch val.(type) {
			case *int64:
				out[i] = fmt.Sprint(*val.(*int64))
			case *int:
				out[i] = fmt.Sprint(*val.(*int))
			case *string:
				out[i] = *val.(*string)
			case *[]byte:
				out[i] = string(*val.(*[]byte))
			default:
				log.Printf("DoStreamingQuery(): Type %+v not handled!", reflect.TypeOf(val))
				out[i] = "?"
			}
		}
		w.Write(out)
	}
	return nil
}

func buildResultsetScanContainers(cols []*Column) (pointers []interface{}) {
	pointers = make([]interface{}, len(cols)+1)
	for i := range pointers {
		if i == 0 { //UID
			var out int
			pointers[i] = &out
		} else if cols[i-1].Datatype == INT || cols[i-1].Datatype == UINT {
			var out int64
			pointers[i] = &out
		} else if cols[i-1].Datatype == STR {
			var out string
			pointers[i] = &out
		} else if cols[i-1].Datatype == BLOB {
			var out []byte
			pointers[i] = &out
		} else if cols[i-1].Datatype == TIME {
			var out time.Time
			pointers[i] = &out
		} else {
			panic("Havent implemented type " + ColDatatype(cols[i-1].Datatype))
		}
	}
	return
}

// DoDelete implements all the logic to delete a datastore.
func DoDelete(ctx context.Context, ds *Datastore, db *sql.DB) error {
	cols, err := GetColumns(ctx, ds.UID, db)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, col := range cols {
		_, err = tx.ExecContext(ctx, `DELETE FROM datastore_col_meta WHERE id()=$1;`, col.UID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM datastore_meta WHERE id()=$1;`, ds.UID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("DROP TABLE ds_" + strconv.Itoa(ds.UID) + ";")
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// DoCreate implements all the logic to create a datastore.
func DoCreate(ctx context.Context, ds *Datastore, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	storeUID, err := makeDatastore(ctx, tx, ds, db)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, col := range ds.Cols {
		err = makeColumn(ctx, tx, storeUID, col, db)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	createQuery := "CREATE TABLE ds_" + strconv.Itoa(storeUID) + " (\n"
	for _, col := range ds.Cols {
		createQuery += columnName(col.Name) + " " + ColDatatype(col.Datatype) + " NOT NULL,\n"
	}
	createQuery += ");\n"

	_, err = tx.Exec(createQuery)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
