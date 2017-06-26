package integration

import (
	"context"
	"database/sql"
	"time"
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
    integration_parent INT NOT NULL,
	  run_id STRING NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
    kind STRING NOT NULL,
    level INT NOT NULL,
    datatype INT,
    value STRING,
	);

  CREATE INDEX IF NOT EXISTS integration_log_by_parent_id ON integration_log(integration_parent);
  CREATE INDEX IF NOT EXISTS integration_log_by_run_id ON integration_log(run_id);
  CREATE INDEX IF NOT EXISTS integration_log_by_time ON integration_log(created_at);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
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

// GetLogsForRunnable is called to get all logs for a runnable.
func GetLogsForRunnable(ctx context.Context, runnableUID int, newerThan time.Time, db *sql.DB) ([]*Log, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), integration_parent, run_id, created_at, kind, level, datatype, value FROM integration_log WHERE integration_parent = $1 AND created_at > $2;
	`, runnableUID, newerThan)
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
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
    INSERT INTO
			integration_log (integration_parent, run_id, kind, level, datatype, value)
			VALUES (
				$1, $2,	$3, $4, $5, $6
			);
	`, log.ParentUID, log.RunID, log.Kind, log.Level, log.Datatype, log.Value)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
