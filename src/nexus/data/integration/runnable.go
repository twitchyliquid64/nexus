package integration

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/util"
	"time"
)

// Kinds of integrations.
const (
	KindRunnable = "Runnable"
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
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  owner_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  name varchar(128) NOT NULL,
    content TEXT DEFAULT "",
		max_retention INT NOT NULL DEFAULT 21
	);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return t.migrateMaxRetentionColumn(ctx, db)
}

func (t *Table) migrateMaxRetentionColumn(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "SELECT max_retention FROM integration_runnable LIMIT 1;")
	if err == nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE integration_runnable ADD COLUMN max_retention INT NOT NULL DEFAULT 21;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *Table) Forms() []*util.FormDescriptor {
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

	MaxRetention int

	Triggers []*Trigger
}

// GetAllRunnable returns all the runnables in the system.
func GetAllRunnable(ctx context.Context, db *sql.DB) ([]*Runnable, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, created_at, name, content, max_retention FROM integration_runnable;`)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Runnable
	for res.Next() {
		var o Runnable
		o.Kind = KindRunnable
		if err := res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Name, &o.Content, &o.MaxRetention); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// GetRunnable returns a runnable by its UID
func GetRunnable(ctx context.Context, uid int, db *sql.DB) (*Runnable, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, created_at, name, content, max_retention FROM integration_runnable WHERE rowid = ?;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, errors.New("Could not find runnable with that UID")
	}

	var o Runnable
	o.Kind = KindRunnable
	return &o, res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Name, &o.Content, &o.MaxRetention)
}

// GetAllForUser is called to get all runnables owned by a given user uid.
func GetAllForUser(ctx context.Context, ownerUID int, db *sql.DB) ([]*Runnable, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, created_at, name, content, max_retention FROM integration_runnable WHERE owner_uid = ?;
	`, ownerUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Runnable
	for res.Next() {
		var o Runnable
		o.Kind = KindRunnable
		if err := res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Name, &o.Content, &o.MaxRetention); err != nil {
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
			integration_runnable (owner_uid, name, content, max_retention)
			VALUES (
				?, ?, ?, ?
			);`, r.OwnerID, r.Name, r.Content, r.MaxRetention)
	if err != nil {
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func editRunnable(ctx context.Context, tx *sql.Tx, r *Runnable, db *sql.DB) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE integration_runnable
			SET name=?, content=?, max_retention = ?
			WHERE rowid = ?;`, r.Name, r.Content, r.MaxRetention, r.UID)
	return err
}

// SaveCode updates just the content of a runnable with the given UID.
func SaveCode(ctx context.Context, UID int, code string, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		UPDATE integration_runnable
			SET content=?
			WHERE rowid = ?;
	`, code, UID)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
