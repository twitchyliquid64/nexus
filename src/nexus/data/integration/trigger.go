package integration

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/dlock"
	util "nexus/data/util"
	"time"
)

// TriggerTable (triggers) implements the databaseTable interface.
type TriggerTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *TriggerTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS integration_trigger (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    integration_parent INT NOT NULL,
	  owner_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  name varchar(128) NOT NULL,
    kind varchar(32) NOT NULL,

		val1 varchar(2048),
		val2 varchar(2048)
	);

  CREATE INDEX IF NOT EXISTS integration_trigger_by_parent_id ON integration_trigger(integration_parent);
	`)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *TriggerTable) Forms() []*util.FormDescriptor {
	return nil
}

// Trigger is the DAO representing a runnables triggers.
type Trigger struct {
	UID       int
	ParentUID int
	OwnerUID  int
	CreatedAt time.Time
	Name      string
	Kind      string

	Val1 string
	Val2 string
}

// GetTriggerByUID returns a specific Trigger DAO.
func GetTriggerByUID(ctx context.Context, uid int, db *sql.DB) (*Trigger, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT rowid, integration_parent, owner_uid, created_at, name, kind, val1, val2 FROM integration_trigger WHERE rowid = ?;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, errors.New("no trigger with that UID")
	}

	var o Trigger
	return &o, res.Scan(&o.UID, &o.ParentUID, &o.OwnerUID, &o.CreatedAt, &o.Name, &o.Kind, &o.Val1, &o.Val2)
}

// GetTriggersForRunnable is called to get all triggers for a runnable.
func GetTriggersForRunnable(ctx context.Context, runnableUID int, db *sql.DB) ([]*Trigger, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT rowid, integration_parent, owner_uid, created_at, name, kind, val1, val2 FROM integration_trigger WHERE integration_parent = ?;
	`, runnableUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Trigger
	for res.Next() {
		var o Trigger
		if err := res.Scan(&o.UID, &o.ParentUID, &o.OwnerUID, &o.CreatedAt, &o.Name, &o.Kind, &o.Val1, &o.Val2); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// GetAllTriggers is called to get all triggers.
func GetAllTriggers(ctx context.Context, db *sql.DB) ([]*Trigger, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT rowid, integration_parent, owner_uid, created_at, name, kind, val1, val2 FROM integration_trigger;
	`)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Trigger
	for res.Next() {
		var o Trigger
		if err := res.Scan(&o.UID, &o.ParentUID, &o.OwnerUID, &o.CreatedAt, &o.Name, &o.Kind, &o.Val1, &o.Val2); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

func makeTrigger(ctx context.Context, tx *sql.Tx, t *Trigger, db *sql.DB) (int, error) {
	x, err := tx.ExecContext(ctx, `
		INSERT INTO
			integration_trigger (integration_parent, owner_uid, name, kind, val1, val2)
			VALUES (
				?, ?, ?, ?, ?, ?
			);`, t.ParentUID, t.OwnerUID, t.Name, t.Kind, t.Val1, t.Val2)
	if err != nil {
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func editTrigger(ctx context.Context, tx *sql.Tx, t *Trigger, db *sql.DB) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE integration_trigger
			SET name=?, kind=?, val1=?, val2=?
			WHERE rowid = ?;`, t.Name, t.Kind, t.Val1, t.Val2, t.UID)
	return err
}
