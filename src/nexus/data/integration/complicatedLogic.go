package integration

import (
	"context"
	"database/sql"
	"time"
)

// DoLogsCleanup deletes old log entries.
func DoLogsCleanup(ctx context.Context, days int, db *sql.DB) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	l, err := tx.ExecContext(ctx, `DELETE FROM integration_log WHERE created_at < $1;`, time.Now().AddDate(0, 0, -days))
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	num, err := l.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return num, tx.Commit()
}

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

// DoEditRunnable applies an edit to a runnable and its triggers.
func DoEditRunnable(ctx context.Context, r *Runnable, db *sql.DB) error {
	currentTriggers, err := GetTriggersForRunnable(ctx, r.UID, db)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	err = editRunnable(ctx, tx, r, db)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, t := range r.Triggers {
		if t.UID == 0 { //create
			_, err := makeTrigger(ctx, tx, t, db)
			if err != nil {
				tx.Rollback()
				return err
			}
		} else { //edit
			err = editTrigger(ctx, tx, t, db)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	for _, earlierExistingTrigger := range currentTriggers {
		if !inTriggerSet(r.Triggers, earlierExistingTrigger.UID) {
			_, err = tx.ExecContext(ctx, `DELETE FROM integration_trigger WHERE id()=$1;`, earlierExistingTrigger.UID)
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return tx.Commit()
}

func inTriggerSet(triggers []*Trigger, uid int) bool {
	for _, t := range triggers {
		if t.UID == uid {
			return true
		}
	}
	return false
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

	_, err = tx.ExecContext(ctx, `DELETE FROM integration_stddata WHERE integration_parent=$1;`, uid)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM integration_log WHERE integration_parent=$1;`, uid)
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
