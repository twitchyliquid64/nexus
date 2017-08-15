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

func findColumnInfo(cols []*Column, name string) *Column {
	for _, col := range cols {
		if col.Name == name {
			return col
		}
	}
	return nil
}

// InsertRow inserts a row into the named database. Invoked by integrations.
// TODO: Add documentation to integration editor
func InsertRow(ctx context.Context, dsID int, rowData map[string]interface{}, db *sql.DB) (int64, error) {
	cols, err := GetColumns(ctx, dsID, db)
	if err != nil {
		return 0, err
	}

	// construct the query & parameters
	queryStr := "INSERT INTO ds_" + strconv.Itoa(dsID) + " ("
	var parameters []interface{}

	accumulator := 0
	for fieldName, fieldValue := range rowData {
		col := findColumnInfo(cols, fieldName)
		if col == nil {
			return 0, fmt.Errorf("cannot find column named %q", fieldName)
		}
		t := reflect.TypeOf(fieldValue)
		if t != nil && t.Kind() == reflect.Float64 && col.Datatype != FLOAT {
			if col.Datatype == INT {
				fieldValue = int64(fieldValue.(float64))
			} else {
				return 0, fmt.Errorf("Bad datatype for non-float column %q, got %s", col.Name, t.Kind().String())
			}
		}
		if t != nil && t.Kind() == reflect.Int64 && col.Datatype != INT {
			if col.Datatype == FLOAT {
				fieldValue = float64(fieldValue.(int64))
			} else {
				return 0, fmt.Errorf("Bad datatype for non-int column %q, got %s", col.Name, t.Kind().String())
			}
		}
		if t != nil && t.Kind() == reflect.String && col.Datatype != STR {
			return 0, fmt.Errorf("Bad datatype for non-string column %q, got %s", col.Name, t.Kind().String())
		}
		// TODO: Handle the TIME and BLOB types.

		queryStr += columnName(col.Name)
		parameters = append(parameters, fieldValue)

		if accumulator < (len(rowData) - 1) {
			queryStr += ", "
		}
		accumulator++
	}
	queryStr += ") VALUES ("
	for i := range parameters {
		queryStr += "?"
		if i < (len(rowData) - 1) {
			queryStr += ", "
		}
	}
	queryStr += ")"

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	r, err := tx.Exec(queryStr + ";", parameters...)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	rowID, err := r.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return rowID, tx.Commit()
}

// DoStreamingInsert takes input as a CSV and inserts it to a database.
// TODO: Refactor
func DoStreamingInsert(ctx context.Context, data io.Reader, dsID int, colIDs []int, db *sql.DB) error {
	insertCols := make([]*Column, len(colIDs))
	cols, err := GetColumns(ctx, dsID, db)
	if err != nil {
		return err
	}

	// make the querystring
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
		queryStr += "?"
		if i < len(colIDs)-1 {
			queryStr += ", "
		}
	}
	queryStr += ");"
	log.Printf("Streaming insert querystring: %s\n", queryStr)

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	r := csv.NewReader(data)
	inContainers := make([]interface{}, len(colIDs))

	// iterate through the source rows
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tx.Rollback()
			return err
		}

		log.Println(row) //TODO: Remove
		for i := range inContainers {
			if insertCols[i].Datatype == INT || insertCols[i].Datatype == UINT {
				v, _ := strconv.Atoi(row[i])
				inContainers[i] = v
			} else if insertCols[i].Datatype == BLOB {
				inContainers[i] = []byte(row[i])
			} else if insertCols[i].Datatype == TIME {
				v, e := strconv.Atoi(row[i])
				if e == nil {
					inContainers[i] = time.Unix(int64(v), 0)
				} else {
					inContainers[i], _ = time.Parse(time.RFC1123, row[i])
				}

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
			switch v := val.(type) {
			case *int64:
				out[i] = fmt.Sprint(*val.(*int64))
			case *int:
				out[i] = fmt.Sprint(*val.(*int))
			case *string:
				out[i] = *val.(*string)
			case *[]byte:
				out[i] = string(*val.(*[]byte))
			case *time.Time:
				out[i] = strconv.Itoa(int(v.Unix()))
			default:
				log.Printf("DoStreamingQuery(): Type %+v not handled!", reflect.TypeOf(val))
				out[i] = "?T?"
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

	_, err = tx.ExecContext(ctx, `DELETE FROM datastore_grant WHERE ds_uid=?;`, ds.UID)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, col := range cols {
		_, err = tx.ExecContext(ctx, `DELETE FROM datastore_col_meta WHERE rowid=?;`, col.UID)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM datastore_meta WHERE rowid=?;`, ds.UID)
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

	createQuery := "CREATE TABLE ds_" + strconv.Itoa(storeUID) + " ("
	for i, col := range ds.Cols {
		createQuery += columnName(col.Name) + " " + ColDatatype(col.Datatype) + " NOT NULL"
		if i < (len(ds.Cols) - 1) {
			createQuery += ", "
		}
	}
	createQuery += ");\n"
	log.Println("[Datastore] Create query:", createQuery)

	_, err = tx.Exec(createQuery)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
