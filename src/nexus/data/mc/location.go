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

	CREATE INDEX IF NOT EXISTS mc_entity_locations_entity_time ON mc_entity_locations(entity_uid, created_at);
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

// LocationsCountForEntityRecent returns the number of location updates for the given entity in the last 24 hours.
func LocationsCountForEntityRecent(ctx context.Context, id int, db *sql.DB) (int, error) {
	res, err := db.QueryContext(ctx, `
		SELECT COUNT(*) FROM mc_entity_locations WHERE entity_uid = ? AND created_at > date('now', '-1 day');
	`, id)
	if err != nil {
		return 0, err
	}
	defer res.Close()

	if !res.Next() {
		return 0, nil
	}

	var o int
	return o, res.Scan(&o)
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
