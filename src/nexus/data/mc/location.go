package mc

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"time"
)

// LocationTable (mc_entity_locations) implements the databaseTable interface.
type LocationTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *LocationTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS mc_entity_locations (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    lat REAL NOT NULL,
    lon REAL NOT NULL,
    kph REAL NOT NULL,
    acc INT NOT NULL,
    course INT NOT NULL
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

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *LocationTable) Forms() []*util.FormDescriptor {
	return nil
}

// Location represents a location update pushed from the entity.
type Location struct {
	UID          int
	EntityKeyUID int
	CreatedAt    time.Time

	Lat, Lon, Kph float64
	Accuracy      int
	Course        int
}

// CreateLocation creates a location entry.
func CreateLocation(ctx context.Context, s *Location, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	e, err := tx.Exec(`
      INSERT INTO
        mc_entity_locations (entity_uid, lat, lon, kph, acc, course)
        VALUES (
          ?, ?, ?, ?, ?, ?
        );
    `, s.EntityKeyUID, s.Lat, s.Lon, s.Kph, s.Accuracy, s.Course)
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
