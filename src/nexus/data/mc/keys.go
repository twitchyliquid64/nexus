package mc

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"nexus/data/session"
	"nexus/data/util"
	"os"
	"strconv"
	"time"
)

// APIKeyTable (mc_entity_keys) implements the databaseTable interface.
type APIKeyTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *APIKeyTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS mc_entity_keys (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    key varchar(128) NOT NULL,
		name varchar(256) NOT NULL,
    kind varchar(128)
	);

  CREATE INDEX IF NOT EXISTS mc_entity_keys_by_key ON mc_entity_keys(key);
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
func (t *APIKeyTable) Forms() []*util.FormDescriptor {
	randomAPIKey, err := session.GenerateRandomString(8)
	if err != nil {
		panic(err) // random source failure, we should crash out
	}
	randomAPIKey = url.QueryEscape(randomAPIKey)

	return []*util.FormDescriptor{
		&util.FormDescriptor{
			SettingsTitle: "Entity API Keys",
			ID:            "mcApiKeys",
			Desc:          "API keys for operational entities.",
			Forms: []*util.ActionDescriptor{
				&util.ActionDescriptor{
					Name:   "New API Key",
					ID:     "mcApiKeys_add",
					IcoStr: "library_add",
					Fields: []*util.Field{
						&util.Field{
							Name: "Type",
							ID:   "kind",
							Kind: "select",
							SelectOptions: map[string]string{
								"phone":             "Phone",
								"static_autonomous": "Installation",
							},
						},
						&util.Field{
							Name:              "Name",
							ID:                "name",
							Kind:              "text",
							ValidationPattern: "[0-9A-Za-z_\\s]{1,256}",
						},
						&util.Field{
							Name:              "APIKey",
							ID:                "key",
							Kind:              "text",
							ValidationPattern: "^[a-zA-Z0-9_-%]{1,18}",
							Val:               randomAPIKey,
						},
					},
					OnSubmit: t.addKeySubmitHandler,
				},
			},
			Tables: []*util.TableDescriptor{
				&util.TableDescriptor{
					Name: "API Keys",
					ID:   "mcApiKeys_table",
					Cols: []string{"#", "Type", "Name", "Key"},
					Actions: []*util.TableAction{
						&util.TableAction{
							Action:       "Delete",
							MaterialIcon: "delete",
							ID:           "mcApiKeys_table_delete",
							Handler:      t.deleteKeyHandler,
						},
					},
					FetchContent: func(ctx context.Context, userID int, db *sql.DB) ([]interface{}, error) {
						data, err := GetEntityKeysForUser(ctx, userID, db)
						if err != nil {
							return nil, err
						}
						out := make([]interface{}, len(data))
						for i, s := range data {
							out[i] = []interface{}{s.UID, s.Kind, s.Name, s.Key}
						}
						return out, nil
					},
				},
			},
		},
	}
}

func (t *APIKeyTable) deleteKeyHandler(rowID, formID, actionUID string, userID int, db *sql.DB) error {
	uid, err := strconv.Atoi(rowID)
	if err != nil {
		return nil
	}
	key, err := GetEntityKeyByUID(context.Background(), uid, db)
	if err != nil {
		return nil
	}
	if key.OwnerID != userID {
		return errors.New("You do not have permission to modify that source")
	}
	return DeleteAPIKey(context.Background(), key.UID, db)
}

// Called on the submission of the form 'New API Key'
func (t *APIKeyTable) addKeySubmitHandler(ctx context.Context, vals map[string]string, userID int, db *sql.DB) error {
	if vals["kind"] != "phone" && vals["kind"] != "static_autonomous" {
		return errors.New("Invalid type")
	}

	_, err := CreateAPIKey(ctx, &APIKey{
		OwnerID: userID,
		Kind:    vals["kind"],
		Key:     vals["key"],
		Name:    vals["name"],
	}, db)
	return err
}

//APIKey represents an APIKey object in persistant storage.
type APIKey struct {
	UID       int
	OwnerID   int
	CreatedAt time.Time
	Key       string
	Kind      string
	Name      string
}

// GetEntityKey returns a APIKey with the given key.
func GetEntityKey(ctx context.Context, key string, db *sql.DB) (*APIKey, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, created_at, key, kind, name FROM mc_entity_keys WHERE key = ?;
	`, key)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, os.ErrNotExist
	}

	var o APIKey
	return &o, res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Key, &o.Kind, &o.Name)
}

// GetEntityKeyByUID returns a APIKey with the given UID.
func GetEntityKeyByUID(ctx context.Context, id int, db *sql.DB) (*APIKey, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, created_at, key, kind, name FROM mc_entity_keys WHERE rowid = ?;
	`, id)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, os.ErrNotExist
	}

	var o APIKey
	return &o, res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Key, &o.Kind, &o.Name)
}

// GetEntityKeysForUser returns all APIKeys owned by a specific user.
func GetEntityKeysForUser(ctx context.Context, uid int, db *sql.DB) ([]APIKey, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, owner_uid, created_at, key, kind, name FROM mc_entity_keys WHERE owner_uid = ?;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []APIKey
	for res.Next() {
		var o APIKey
		if err := res.Scan(&o.UID, &o.OwnerID, &o.CreatedAt, &o.Key, &o.Kind, &o.Name); err != nil {
			return nil, err
		}
		output = append(output, o)
	}
	return output, nil
}

// CreateAPIKey creates a APIKey entry.
func CreateAPIKey(ctx context.Context, key *APIKey, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	e, err := tx.Exec(`
      INSERT INTO
        mc_entity_keys (owner_uid, key, kind, name)
        VALUES (
          ?, ?, ?, ?
        );
    `, key.OwnerID, key.Key, key.Kind, key.Name)
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

// DeleteAPIKey removes a APIKey.
func DeleteAPIKey(ctx context.Context, id int, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM
			mc_entity_keys WHERE rowid = ?;`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}
