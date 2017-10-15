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
func LocationsCountForEntityRecent(ctx context.Context, id int, db *sql.DB) (int, time.Time, error) {
	res, err := db.QueryContext(ctx, `
		SELECT * FROM
			(SELECT COUNT(*) FROM mc_entity_locations WHERE entity_uid = ? AND created_at > date('now', '-1 day')),
			(SELECT created_at FROM mc_entity_locations WHERE entity_uid = ? ORDER BY created_at DESC LIMIT 1);
	`, id, id)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer res.Close()

	if !res.Next() {
		return 0, time.Time{}, nil
	}

	var o int
	var t time.Time
	return o, t, res.Scan(&o, &t)
}

// ListLocation returns the locations for an entity within the two times.
func ListLocation(ctx context.Context, uid int, from, to time.Time, db *sql.DB) ([]Location, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, entity_uid, created_at, lat, lon, kph, acc, course
			FROM mc_entity_locations WHERE entity_uid = ? AND created_at > ? AND created_at < ? ORDER BY created_at DESC;`, uid, from, to)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []Location
	for res.Next() {
		var out Location
		if err := res.Scan(&out.UID, &out.EntityKeyUID, &out.CreatedAt, &out.Lat, &out.Lon, &out.Kph, &out.Accuracy, &out.Course); err != nil {
			return nil, err
		}
		output = append(output, out)
	}
	return output, nil
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
