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
    is_heartbeat BOOL NOT NULL
	);

	CREATE INDEX IF NOT EXISTS mc_entity_statuses_entity_time ON mc_entity_statuses(entity_uid, created_at);

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
func (t *StatusTable) Forms() []*util.FormDescriptor {
	return nil
}

// Status represents a status update pushed from the entity.
type Status struct {
	UID          int
	EntityKeyUID int
	CreatedAt    time.Time
	Status       string
	BatteryLevel int
	IsHeartbeat  bool
}

// CreateStatus creates a status entry.
func CreateStatus(ctx context.Context, s *Status, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	e, err := tx.Exec(`
      INSERT INTO
        mc_entity_statuses (entity_uid, status, bat, is_heartbeat)
        VALUES (
          ?, ?, ?, ?
        );
    `, s.EntityKeyUID, s.Status, s.BatteryLevel, s.IsHeartbeat)
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
