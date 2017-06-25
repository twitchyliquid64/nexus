package integration

import (
	"context"
	"database/sql"
)

// DoCreateRunnable implements all the logic to create a runnable and its triggers.
func DoCreateRunnable(ctx context.Context, ds *Runnable, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	pUID, err := makeRunnable(ctx, tx, ds, db)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, trigger := range ds.Triggers {
		trigger.OwnerUID = ds.OwnerID
		trigger.ParentUID = pUID
		_, err := makeTrigger(ctx, tx, trigger, db)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// DoDeleteRunnable implements all the logic to delete a runnable and its triggers.
func DoDeleteRunnable(ctx context.Context, uid int, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM integration_trigger WHERE integration_parent=$1;`, uid)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM integration_runnable WHERE id()=$1;`, uid)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
