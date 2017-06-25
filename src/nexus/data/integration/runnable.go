package integration

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Table (runnable) implements the databaseTable interface.
type Table struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *Table) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS integration_runnable (
	  owner_uid INT NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
	  name STRING NOT NULL,
    content STRING DEFAULT "",
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

// Runnable is a DAO representing a integration/script.
type Runnable struct {
	UID       int
	Name      string
	Kind      string
	OwnerID   int
	CreatedAt time.Time
	Content   string

	Triggers []*Trigger
}

// GetRunnable returns a runnable by its UID
func GetRunnable(ctx context.Context, uid int, db *sql.DB) (*Runnable, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), owner_uid, created_at, name, content FROM integration_runnable WHERE id() = $1;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, errors.New("Could not find runnable with that UID")
	}

	var o Runnable
	o.Kind = "Runnable"
	return &o, res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Name, &o.Content)
}

// GetAllForUser is called to get all runnables owned by a given user uid.
func GetAllForUser(ctx context.Context, ownerUID int, db *sql.DB) ([]*Runnable, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), owner_uid, created_at, name, content FROM integration_runnable WHERE owner_uid = $1;
	`, ownerUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Runnable
	for res.Next() {
		var o Runnable
		o.Kind = "Runnable"
		if err := res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Name, &o.Content); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// makeRunnable creates a runnable entry only - triggers should be created separately.
func makeRunnable(ctx context.Context, tx *sql.Tx, r *Runnable, db *sql.DB) (int, error) {
	x, err := tx.ExecContext(ctx, `
		INSERT INTO
			integration_runnable (owner_uid, name, content)
			VALUES (
				$1, $2,	$3
			);`, r.OwnerID, r.Name, r.Content)
	if err != nil {
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}
