package messaging

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"nexus/data/dlock"
	"nexus/data/util"
	"nexus/metrics"
	"os"
	"strconv"
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
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  name varchar(128) NOT NULL,
		owner_id int NOT NULL,
    kind varchar(16) NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  remote BOOLEAN NOT NULL DEFAULT 0,
    details_json TEXT
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
	return []*util.FormDescriptor{
		&util.FormDescriptor{
			SettingsTitle: "Messenger Sources",
			ID:            "messengerUserSources",
			Desc:          "Sources are integrations with remote IM sources such as slack or IRC.",
			Forms: []*util.ActionDescriptor{
				&util.ActionDescriptor{
					Name:   "New Source",
					ID:     "messenger_source_add",
					IcoStr: "message",
					Fields: []*util.Field{
						&util.Field{
							Name: "Source Type",
							ID:   "kind",
							Kind: "select",
							SelectOptions: map[string]string{
								Slack: "Slack",
								IRC:   "IRC",
							},
						},
						&util.Field{
							Name: "Name",
							ID:   "name",
							Kind: "text",
						},
						&util.Field{
							Name: "Details struct (JSON)",
							ID:   "details_json",
							Kind: "text",
						},
					},
					OnSubmit: t.addSourceSubmitHandler,
				},
			},
			Tables: []*util.TableDescriptor{
				&util.TableDescriptor{
					Name: "Existing Sources",
					ID:   "messenger_existing_sources",
					Cols: []string{"#", "Name", "Type"},
					Actions: []*util.TableAction{
						&util.TableAction{
							Action:       "Delete",
							MaterialIcon: "delete",
							ID:           "messenger_existing_sources_delete",
							Handler:      t.deleteSourceActionHandler,
						},
					},
					FetchContent: func(ctx context.Context, userID int, db *sql.DB) ([]interface{}, error) {
						data, err := GetAllSourcesForUser(ctx, userID, db)
						if err != nil {
							return nil, err
						}
						out := make([]interface{}, len(data))
						for i, s := range data {
							out[i] = []interface{}{s.UID, s.Name, s.Kind}
						}
						return out, nil
					},
				},
			},
		},
	}
}

func (t *SourceTable) deleteSourceActionHandler(rowID, formID, actionUID string, userID int, db *sql.DB) error {
	uid, err := strconv.Atoi(rowID)
	if err != nil {
		return nil
	}
	src, _, err := GetSourceByUID(context.Background(), uid, db)
	if err != nil {
		return nil
	}
	if src.OwnerID != userID {
		return errors.New("You do not have permission to modify that source")
	}
	return DeleteSource(context.Background(), src.UID, db)
}

// Called on the submission of the form 'New Source'
func (t *SourceTable) addSourceSubmitHandler(ctx context.Context, vals map[string]string, userID int, db *sql.DB) error {
	if vals["kind"] != Slack && vals["kind"] != IRC {
		return errors.New("Invalid source type")
	}

	var d map[string]string
	err := json.Unmarshal([]byte(vals["details_json"]), &d)
	if err != nil {
		return err
	}

	err = AddSource(ctx, Source{
		OwnerID: userID,
		Name:    vals["name"],
		Kind:    vals["kind"],
		Details: d,
	}, db)
	return err
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

// DeleteSource removes a source.
func DeleteSource(ctx context.Context, id int, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM
			messaging_source WHERE rowid = ?;`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// AddSource creates a new messaging source.
func AddSource(ctx context.Context, src Source, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

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
		VALUES (?, ?, ?, ?, ?);
	`, src.Name, src.OwnerID, src.Kind, src.Remote, string(details))
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// GetAllSourcesForUser is called to get all messaging sources for a given uid.
func GetAllSourcesForUser(ctx context.Context, uid int, db *sql.DB) ([]*Source, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()
	defer metrics.GetMessagingSourcesUIDDbTime.Time(time.Now())

	res, err := db.QueryContext(ctx, `
		SELECT rowid, name, owner_id, kind, remote, created_at, details_json FROM messaging_source WHERE owner_id = ?;
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
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `SELECT rowid, name, owner_id, kind, remote, created_at, details_json FROM messaging_source;`)
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

// GetSourceByUID returns a source with the given UID.
func GetSourceByUID(ctx context.Context, uid int, db *sql.DB) (*Source, string, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT rowid, name, owner_id, kind, remote, created_at, details_json FROM messaging_source WHERE rowid = ?;
	`, uid)
	if err != nil {
		return nil, "", err
	}
	defer res.Close()

	if !res.Next() {
		return nil, "", os.ErrNotExist
	}

	var o Source
	var detailsJSONStr string
	return &o, detailsJSONStr, res.Scan(&o.UID, &o.Name, &o.OwnerID, &o.Kind, &o.Remote, &o.CreatedAt, &detailsJSONStr)
}
