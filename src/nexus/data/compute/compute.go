package compute

import (
	"context"
	"database/sql"
	"nexus/data/dlock"
	"nexus/data/util"
	"time"
)

// InstanceTable (compute_instances) implements the databaseTable interface.
type InstanceTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *InstanceTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS compute_instances (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    kind varchar(128) NOT NULL,
		name varchar(64) NOT NULL,
    id varchar(128) NOT NULL,
    metadata BLOB NOT NULL,
    auth_credentials BLOB NOT NULL,
		ssh_data BLOB NOT NULL
	);

  CREATE INDEX IF NOT EXISTS compute_instances_owner ON compute_instances(owner_uid);
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
func (t *InstanceTable) Forms() []*util.FormDescriptor {
	return nil
}

// Instance represents the information associated with a cloud instance.
type Instance struct {
	UID       int
	OwnerID   int
	CreatedAt time.Time
	ExpiresAt time.Time
	Kind      string
	Name      string
	ID        string
	Metadata  string
	Auth      string
	SSH       string
}

// GetAll returns all Instances.
func GetAll(ctx context.Context, db *sql.DB) ([]Instance, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, created_at, expires_at, kind, name, id, metadata, auth_credentials, ssh_data FROM compute_instances;`)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []Instance
	for res.Next() {
		var o Instance
		if err := res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.ExpiresAt, &o.Kind, &o.Name, &o.ID, &o.Metadata, &o.Auth, &o.SSH); err != nil {
			return nil, err
		}
		output = append(output, o)
	}
	return output, nil
}

// New creates a new instance.
func New(ctx context.Context, i Instance, db *sql.DB) (int64, error) {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	e, err := tx.Exec(`
        INSERT INTO
          compute_instances (owner_uid, expires_at, kind, name, id, metadata, auth_credentials, ssh_data)
          VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?
          );
      `, i.OwnerID, i.ExpiresAt, i.Kind, i.Name, i.ID, i.Metadata, i.Auth, i.SSH)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	lastID, err := e.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return lastID, tx.Commit()
}

// Delete removes an instance from the database.
func Delete(ctx context.Context, uid int, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		DELETE FROM
			compute_instances
		WHERE rowid = ?;
	`, uid)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
