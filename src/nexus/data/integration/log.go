package integration

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"nexus/metrics"
	"strconv"
	"time"
)

// types of log levels
const (
	LevelInfo int = iota
	LevelWarning
	LevelError
)

// kinds of messages
const (
	KindLog            = "log"
	KindControlLog     = "control"
	KindStructuredData = "data"
	KindJSONData       = "json"
)

// datatypes
const (
	DatatypeUnstructured int = iota
	DatatypeString
	DatatypeInt
	DatatypeStartInfo
	DatatypeEndInfo
	DatatypeTrace
)

// LogTable (log) implements the databaseTable interface.
type LogTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *LogTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS integration_log (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    integration_parent INT NOT NULL,
	  run_id varchar(64) NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    kind varchar(32) NOT NULL,
    level INT NOT NULL,
    datatype INT,
    value TEXT
	);

  CREATE INDEX IF NOT EXISTS integration_log_combined ON integration_log(integration_parent, run_id, created_at);
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
func (t *LogTable) Forms() []*util.FormDescriptor {
	return nil
}

// Log is a DAO representing a single log line emitted by a run of an integration.
type Log struct {
	UID       int
	ParentUID int
	RunID     string
	CreatedAt time.Time
	Kind      string
	Level     int
	Datatype  int
	Value     string
}

// GetRecentRunsForRunnable returns the unique runIDs for a given runnable.
func GetRecentRunsForRunnable(ctx context.Context, runnableUID int, newerThan time.Time, db *sql.DB) ([]string, error) {
	res, err := db.QueryContext(ctx, `
		SELECT DISTINCT run_id FROM integration_log WHERE integration_parent = ? AND created_at > ? ORDER BY created_at DESC LIMIT 50;
	`, runnableUID, newerThan)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []string
	for res.Next() {
		var o string
		if err := res.Scan(&o); err != nil {
			return nil, err
		}
		output = append(output, o)
	}
	return output, nil
}

// GetLogsForRunnable is called to get all logs for a runnable.
func GetLogsForRunnable(ctx context.Context, runnableUID int, newerThan time.Time, offset, limit int, info, prob, sys bool, db *sql.DB) ([]*Log, error) {
	defer metrics.GetLogsByRunnableDbTime.Time(time.Now())
	query := `
	SELECT rowid, integration_parent, run_id, created_at, kind, level, datatype, value FROM integration_log WHERE integration_parent = ? AND created_at > ?
	`
	if !info {
		query += " AND level != " + strconv.Itoa(LevelInfo)
	}
	if !prob {
		query += " AND level != " + strconv.Itoa(LevelWarning)
		query += " AND level != " + strconv.Itoa(LevelError)
	}
	if !sys {
		query += " AND kind != \"" + KindControlLog + "\""
		query += " AND datatype != " + strconv.Itoa(DatatypeStartInfo)
		query += " AND datatype != " + strconv.Itoa(DatatypeEndInfo)
	}

	res, err := db.QueryContext(ctx, query+" ORDER BY created_at ASC LIMIT ? OFFSET ?;", runnableUID, newerThan, limit, offset)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Log
	for res.Next() {
		var o Log
		if err := res.Scan(&o.UID, &o.ParentUID, &o.RunID, &o.CreatedAt, &o.Kind, &o.Level, &o.Datatype, &o.Value); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// GetLogsFilteredByRunnable filters to a specific run.
func GetLogsFilteredByRunnable(ctx context.Context, runnableUID int, newerThan time.Time, runID string, offset, limit int, info, prob, sys bool, db *sql.DB) ([]*Log, error) {
	defer metrics.GetFilteredLogsByRunnableDbTime.Time(time.Now())
	query := `
		SELECT rowid, integration_parent, run_id, created_at, kind, level, datatype, value FROM integration_log
		WHERE run_id = ? AND created_at > ? AND integration_parent = ?
	`
	if !info {
		query += " AND level != " + strconv.Itoa(LevelInfo)
	}
	if !prob {
		query += " AND level != " + strconv.Itoa(LevelWarning)
		query += " AND level != " + strconv.Itoa(LevelError)
	}
	if !sys {
		query += " AND kind != \"" + KindControlLog + "\""
		query += " AND datatype != " + strconv.Itoa(DatatypeStartInfo)
		query += " AND datatype != " + strconv.Itoa(DatatypeEndInfo)
	}

	res, err := db.QueryContext(ctx, query+" ORDER BY created_at ASC LIMIT ? OFFSET ?;", runID, newerThan, runnableUID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Log
	for res.Next() {
		var o Log
		if err := res.Scan(&o.UID, &o.ParentUID, &o.RunID, &o.CreatedAt, &o.Kind, &o.Level, &o.Datatype, &o.Value); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// WriteLog commits a log entry.
func WriteLog(ctx context.Context, log *Log, db *sql.DB) error {
	defer metrics.InsertLogDbTime.Time(time.Now())
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
    INSERT INTO
			integration_log (integration_parent, run_id, kind, level, datatype, value)
			VALUES (
				?, ?, ?, ?, ?, ?
			);
	`, log.ParentUID, log.RunID, log.Kind, log.Level, log.Datatype, log.Value)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
