package datastore

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
)

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

	storeUID, err := MakeDatastore(ctx, tx, ds, db)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, col := range ds.Cols {
		err = MakeColumn(ctx, tx, storeUID, col, db)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	createQuery := "CREATE TABLE ds_" + strconv.Itoa(storeUID) + " (\n"
	for _, col := range ds.Cols {
		createQuery += strings.Replace(col.Name, " ", "_", -1) + "_ " + ColDatatype(col.Datatype) + ",\n"
	}
	createQuery += ");\n"

	_, err = tx.Exec(createQuery)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
