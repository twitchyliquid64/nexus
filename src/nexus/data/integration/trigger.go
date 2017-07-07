package integration

import (
	"context"
	"database/sql"
	"errors"
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
    integration_parent INT NOT NULL,
	  owner_uid INT NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
	  name STRING NOT NULL,
    kind STRING NOT NULL,
	);

  CREATE INDEX IF NOT EXISTS integration_trigger_by_parent_id ON integration_trigger(integration_parent);
	`)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return t.doCheckV2Columns(ctx, db)
}

// check new columns exist, nd do migration if necessary
func (t *TriggerTable) doCheckV2Columns(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	valExists, err := util.ColumnExists(tx, "val1", "integration_trigger")
	if err != nil {
		return err
	}
	if !valExists {
		_, err2 := tx.Exec("ALTER TABLE integration_trigger ADD val1 STRING NOT NULL;")
		if err2 != nil {
			return err2
		}
	}

	valExists2, err := util.ColumnExists(tx, "val2", "integration_trigger")
	if err != nil {
		return err
	}
	if !valExists2 {
		_, err2 := tx.Exec("ALTER TABLE integration_trigger ADD val2 STRING NOT NULL;")
		if err2 != nil {
			return err2
		}
	}
	return tx.Commit()
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
	res, err := db.QueryContext(ctx, `
		SELECT id(), integration_parent, owner_uid, created_at, name, kind, val1, val2 FROM integration_trigger WHERE id() = $1;
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
	res, err := db.QueryContext(ctx, `
		SELECT id(), integration_parent, owner_uid, created_at, name, kind, val1, val2 FROM integration_trigger WHERE integration_parent = $1;
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
	res, err := db.QueryContext(ctx, `
		SELECT id(), integration_parent, owner_uid, created_at, name, kind, val1, val2 FROM integration_trigger;
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
				$1, $2,	$3, $4, $5, $6
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
			SET name=$2, kind=$3, val1=$4, val2=$5
			WHERE id() = $1;`, t.UID, t.Name, t.Kind, t.Val1, t.Val2)
	return err
}
