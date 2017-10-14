package mc

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"time"
)

// StatusTable (mc_entity_status) implements the databaseTable interface.
type StatusTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *StatusTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS mc_entity_statuses (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status varchar(512) NOT NULL,
		bat INT NOT NULL,
    is_heartbeat BOOL NOT NULL,

		is_structured BOOL NOT NULL DEFAULT false,
		additional_data varchar(512) NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS mc_entity_statuses_entity_time ON mc_entity_statuses(entity_uid, created_at);

	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return t.migrateTableColumns(ctx, db)
}

// called on initialization to detect if the table should be migrated to include new columns 'is_structured' and 'additional_data'
func (t *StatusTable) migrateTableColumns(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "SELECT is_structured FROM mc_entity_statuses LIMIT 1;")
	if err == nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE mc_entity_statuses ADD COLUMN is_structured BOOL NOT NULL DEFAULT false;`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE mc_entity_statuses ADD COLUMN additional_data varchar(512) NOT NULL DEFAULT '';`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *StatusTable) Forms() []*util.FormDescriptor {
	return nil
}

// Status represents a status update pushed from the entity.
type Status struct {
	UID          int
	EntityKeyUID int
	CreatedAt    time.Time

	IsStructured   bool
	AdditionalData string

	Status       string
	BatteryLevel int
	IsHeartbeat  bool
}

// RecentStatusInfoForEntity returns information about the latest status.
func RecentStatusInfoForEntity(ctx context.Context, id int, db *sql.DB) (int, time.Time, string, error) {
	res, err := db.QueryContext(ctx, `
		SELECT * FROM
			(SELECT COUNT(*) FROM mc_entity_statuses WHERE entity_uid = ? AND created_at > date('now', '-1 day')),
			(SELECT created_at, status FROM mc_entity_statuses WHERE entity_uid = ? AND is_heartbeat = 0 ORDER BY created_at DESC LIMIT 1);
	`, id, id)
	if err != nil {
		return 0, time.Time{}, "", err
	}
	defer res.Close()

	if !res.Next() {
		return 0, time.Time{}, "", nil
	}

	var o int
	var t time.Time
	var latest string
	return o, t, latest, res.Scan(&o, &t, &latest)
}

// ListStatus returns the statuses for an entity sorted most recent first.
func ListStatus(ctx context.Context, uid, limit, offset int, db *sql.DB) ([]Status, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, entity_uid, created_at, status, bat, is_heartbeat, is_structured, additional_data
			FROM mc_entity_statuses WHERE entity_uid = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;`, uid, limit, offset)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []Status
	for res.Next() {
		var out Status
		if err := res.Scan(&out.UID, &out.EntityKeyUID, &out.CreatedAt, &out.Status, &out.BatteryLevel, &out.IsHeartbeat, &out.IsStructured, &out.AdditionalData); err != nil {
			return nil, err
		}
		output = append(output, out)
	}
	return output, nil
}

// CreateStatus creates a status entry.
func CreateStatus(ctx context.Context, s *Status, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	e, err := tx.Exec(`
      INSERT INTO
        mc_entity_statuses (entity_uid, status, bat, is_heartbeat, is_structured, additional_data)
        VALUES (
          ?, ?, ?, ?, ?, ?
        );
    `, s.EntityKeyUID, s.Status, s.BatteryLevel, s.IsHeartbeat, s.IsStructured, s.AdditionalData)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	lastID, err := e.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return int(lastID), tx.Commit()
}
