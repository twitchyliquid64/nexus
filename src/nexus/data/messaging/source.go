package messaging

import (
	"context"
	"database/sql"
	"encoding/json"
	"nexus/data/util"
	"time"
)

// Kinds of messaging source
const (
	Slack       = "slack"
	IRC         = "irc"
	FbMessenger = "fb_messenger"
)

// SourceTable implements the DataTable interface.
type SourceTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *SourceTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS messaging_source (
	  name STRING NOT NULL,
		owner_id int NOT NULL,
    kind STRING NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
	  remote BOOL NOT NULL DEFAULT FALSE,
    details_json STRING,
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
func (t *SourceTable) Forms() []*util.FormDescriptor {
	return nil
}

// Source is a DAO for messaging_source database rows.
type Source struct {
	UID       int
	Name      string
	Kind      string
	OwnerID   int
	Remote    bool
	CreatedAt time.Time
	Details   map[string]string
}

// AddSource creates a new messaging source.
func AddSource(ctx context.Context, src Source, db *sql.DB) error {
	details, err := json.Marshal(src.Details)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	INSERT INTO
		messaging_source (name, owner_id, kind, remote, details_json)
		VALUES ($1, $2, $3, $4, $5);
	`, src.Name, src.OwnerID, src.Kind, src.Remote, string(details))
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// GetAllSourcesForUser is called to get all messaging sources for a given uid.
func GetAllSourcesForUser(ctx context.Context, uid int, db *sql.DB) ([]*Source, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), name, owner_id, kind, remote, created_at, details_json FROM messaging_source WHERE owner_id = $1;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Source
	for res.Next() {
		var o Source
		var detailsJSONStr string
		if err := res.Scan(&o.UID, &o.Name, &o.OwnerID, &o.Kind, &o.Remote, &o.CreatedAt, &detailsJSONStr); err != nil {
			return nil, err
		}
		if detailsJSONStr != "" {
			err := json.Unmarshal([]byte(detailsJSONStr), &o.Details)
			if err != nil {
				return nil, err
			}
		}
		output = append(output, &o)
	}
	return output, nil
}

// GetAllSources is called to get all messaging sources.
func GetAllSources(ctx context.Context, db *sql.DB) ([]*Source, error) {
	res, err := db.QueryContext(ctx, `SELECT id(), name, owner_id, kind, remote, created_at, details_json FROM messaging_source;`)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Source
	for res.Next() {
		var o Source
		var detailsJSONStr string
		if err := res.Scan(&o.UID, &o.Name, &o.OwnerID, &o.Kind, &o.Remote, &o.CreatedAt, &detailsJSONStr); err != nil {
			return nil, err
		}
		if detailsJSONStr != "" {
			err := json.Unmarshal([]byte(detailsJSONStr), &o.Details)
			if err != nil {
				return nil, err
			}
		}
		output = append(output, &o)
	}
	return output, nil
}
