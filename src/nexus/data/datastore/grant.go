package datastore

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"time"
)

// StoreGrant (datastore_grant) implements the databaseTable interface.
type StoreGrant struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *StoreGrant) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS datastore_grant (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  user_uid int NOT NULL,
	  ds_uid int NOT NULL,
    read_only BOOLEAN NOT NULL DEFAULT 0,
	  created_at TIMESTAMPSTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *StoreGrant) Forms() []*util.FormDescriptor {
	return nil
}

// Grant is the DAO for granting access to a datastore.
type Grant struct {
	UID       int
	UsrUID    int
	DsUID     int
	ReadOnly  bool
	CreatedAt time.Time

	Name string
}

// MakeGrant registers access to the datastore.
func MakeGrant(ctx context.Context, grant *Grant, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	x, err := tx.ExecContext(ctx, `
		INSERT INTO
			datastore_grant (user_uid, ds_uid, read_only)
			VALUES (
				?, ?, ?
			);`, grant.UsrUID, grant.DsUID, grant.ReadOnly)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return int(id), nil
}

// DeleteGrant removes grants corresponding to the given grantID
func DeleteGrant(ctx context.Context, uid int, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM
			datastore_grant
		WHERE rowid = ?;`, uid)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// CheckAccess determines if the user is allowed to perform the given action on the given datastore.
func CheckAccess(ctx context.Context, usrUID, dsUID int, readOnly bool, db *sql.DB) (bool, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid FROM datastore_grant WHERE
    user_uid = ? AND ds_uid = ? AND (read_only = ? OR read_only = 0);`, usrUID, dsUID, readOnly)
	if err != nil {
		return false, err
	}
	defer res.Close()

	if !res.Next() {
		return false, nil
	}
	return true, nil
}

// ListByUser returns all the grants for a given userID.
func ListByUser(ctx context.Context, userUID int, db *sql.DB) ([]*Grant, error) {
	res, err := db.QueryContext(ctx, `
    SELECT
      datastore_grant.rowid, datastore_grant.user_uid, datastore_grant.ds_uid, datastore_grant.read_only, datastore_grant.created_at, datastore_meta.name
    FROM
			datastore_grant, datastore_meta
		WHERE
			datastore_grant.user_uid=? AND datastore_meta.rowid=datastore_grant.ds_uid;`, userUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Grant
	for res.Next() {
		var out Grant
		if err := res.Scan(&out.UID, &out.UsrUID, &out.DsUID, &out.ReadOnly, &out.CreatedAt, &out.Name); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}
