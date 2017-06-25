package integration

import (
	"context"
	"database/sql"
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
}

// GetTriggersForRunnable is called to get all triggers for a runnable.
func GetTriggersForRunnable(ctx context.Context, runnableUID int, db *sql.DB) ([]*Trigger, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), integration_parent, owner_uid, created_at, name, kind FROM integration_trigger WHERE integration_parent = $1;
	`, runnableUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Trigger
	for res.Next() {
		var o Trigger
		if err := res.Scan(&o.UID, &o.ParentUID, &o.OwnerUID, &o.CreatedAt, &o.Name, &o.Kind); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

func makeTrigger(ctx context.Context, tx *sql.Tx, t *Trigger, db *sql.DB) (int, error) {
	x, err := tx.ExecContext(ctx, `
		INSERT INTO
			integration_trigger (integration_parent, owner_uid, name, kind)
			VALUES (
				$1, $2,	$3, $4
			);`, t.ParentUID, t.OwnerUID, t.Name, t.Kind)
	if err != nil {
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}
