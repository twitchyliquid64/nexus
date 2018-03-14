package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"nexus/data/dlock"
	"nexus/data/util"
	"strconv"
	"time"
)

// Different kinds of external apps.
const (
	ExternAppURLKind = 0
	ExternAppJWTKind = 1
)

// ExternalAppsTable (ext_apps) implements the databaseTable interface.
type ExternalAppsTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *ExternalAppsTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS ext_apps (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  uid INT NOT NULL,
	  kind INT NOT NULL DEFAULT 0,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    name VARCHAR(64) NOT NULL,
    icon VARCHAR(128) NOT NULL,
    extra VARCHAR(128) NOT NULL,
    val TEXT
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
func (t *ExternalAppsTable) Forms() []*util.FormDescriptor {
	return []*util.FormDescriptor{
		&util.FormDescriptor{
			SettingsTitle: "External Apps",
			ID:            "usrExtApps",
			Desc:          "External apps are additional links shown the in the users application list.",
			Forms: []*util.ActionDescriptor{
				&util.ActionDescriptor{
					Name:   "New External App (URL)",
					ID:     "usr_ext_app_add_url",
					IcoStr: "add",
					Fields: []*util.Field{
						&util.Field{
							Name:              "Name",
							ID:                "name",
							Kind:              "text",
							ValidationPattern: "[A-Za-z_\\s]{1,18}",
						},
						&util.Field{
							Name:              "Icon",
							ID:                "icon",
							Kind:              "text",
							ValidationPattern: "[A-Za-z_\\s]{1,18}",
						},
						&util.Field{
							Name: "URL",
							ID:   "url",
							Kind: "text",
						},
					},
					OnSubmit: t.addExtAppURLActionHandler,
				},
				&util.ActionDescriptor{
					Name:   "New External App (JWT)",
					ID:     "usr_ext_app_add_jwt",
					IcoStr: "add",
					Fields: []*util.Field{
						&util.Field{
							Name:              "Name",
							ID:                "name",
							Kind:              "text",
							ValidationPattern: "[A-Za-z_\\s]{1,18}",
						},
						&util.Field{
							Name:              "Icon",
							ID:                "icon",
							Kind:              "text",
							ValidationPattern: "[A-Za-z_\\s]{1,18}",
						},
						&util.Field{
							Name: "Post URL",
							ID:   "url",
							Kind: "text",
						},
						&util.Field{
							Name: "HS256 Secret",
							ID:   "secret",
							Kind: "text",
						},
					},
					OnSubmit: t.addExtAppJWTActionHandler,
				},
			},
			Tables: []*util.TableDescriptor{
				&util.TableDescriptor{
					Name: "Existing External Apps",
					ID:   "usr_ext_app_list",
					Cols: []string{"#", "Name", "Icon", "URL", "Kind"},
					Actions: []*util.TableAction{
						&util.TableAction{
							Action:       "Delete",
							MaterialIcon: "delete",
							ID:           "usr_ext_app_list_delete",
							Handler:      t.deleteExtAppActionHandler,
						},
					},
					FetchContent: func(ctx context.Context, userID int, db *sql.DB) ([]interface{}, error) {
						data, err := GetExtAppsForUser(ctx, userID, db)
						if err != nil {
							return nil, err
						}
						out := make([]interface{}, len(data))
						for i, s := range data {
							out[i] = []interface{}{s.UID, s.Name, s.Icon, s.Val, ""}
							switch s.Kind {
							case ExternAppJWTKind:
								out[i].([]interface{})[4] = "JWT"
							case ExternAppURLKind:
								out[i].([]interface{})[4] = "URL"
							}
						}
						return out, nil
					},
				},
			},
		},
	}
}

func (t *ExternalAppsTable) deleteExtAppActionHandler(rowID, formID, actionUID string, userID int, db *sql.DB) error {
	uid, err := strconv.Atoi(rowID)
	if err != nil {
		return err
	}
	src, err := GetExtApp(context.Background(), uid, db)
	if err != nil {
		return err
	}
	if src.UserID != userID {
		return errors.New("You do not have permission to modify that source")
	}
	return DeleteExtApp(context.Background(), src.UID, db)
}

func (t *ExternalAppsTable) addExtAppURLActionHandler(ctx context.Context, vals map[string]string, userID int, db *sql.DB) error {
	return CreateExtApp(ctx, &ExtApp{
		UserID: userID,
		Kind:   ExternAppURLKind,
		Name:   vals["name"],
		Icon:   vals["icon"],
		Val:    vals["url"],
	}, db)
}

func (t *ExternalAppsTable) addExtAppJWTActionHandler(ctx context.Context, vals map[string]string, userID int, db *sql.DB) error {
	enc, err := json.Marshal(struct {
		Secret string `json:"secret"`
	}{
		Secret: vals["secret"],
	})
	if err != nil {
		return err
	}

	return CreateExtApp(ctx, &ExtApp{
		UserID: userID,
		Kind:   ExternAppJWTKind,
		Name:   vals["name"],
		Icon:   vals["icon"],
		Val:    vals["url"],
		Extra:  string(enc),
	}, db)
}

// ExtApp is the DAO for a users external application entry.
type ExtApp struct {
	UID       int
	UserID    int
	CreatedAt time.Time

	Kind        int
	Name        string
	Icon, Extra string

	Val string
}

// GetExtAppsForUser returns a full list of external apps for the given userID.
func GetExtAppsForUser(ctx context.Context, UID int, db *sql.DB) ([]*ExtApp, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `SELECT rowid, uid, kind, created_at, val, name, icon, extra FROM ext_apps WHERE uid=?;`, UID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*ExtApp
	for res.Next() {
		var out ExtApp
		if err := res.Scan(&out.UID, &out.UserID, &out.Kind, &out.CreatedAt, &out.Val, &out.Name, &out.Icon, &out.Extra); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}

// GetExtApp returns an ext_apps entry by its UID/rowid.
func GetExtApp(ctx context.Context, UID int, db *sql.DB) (*ExtApp, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `SELECT rowid, uid, kind, created_at, val, name, icon, extra FROM ext_apps WHERE rowid=?;`, UID)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	if !res.Next() {
		return nil, errors.New("no such entry")
	}

	var out ExtApp
	return &out, res.Scan(&out.UID, &out.UserID, &out.Kind, &out.CreatedAt, &out.Val, &out.Name, &out.Icon, &out.Extra)
}

// CreateExtApp makes a new external app entry.
func CreateExtApp(ctx context.Context, attr *ExtApp, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO
			ext_apps (uid, name, kind, val, icon, extra)
			VALUES (?, ?, ?, ?, ?, ?);`, attr.UserID, attr.Name, attr.Kind, attr.Val, attr.Icon, attr.Extra)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteExtApp removes an external app by ID.
func DeleteExtApp(ctx context.Context, id int, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM
			ext_apps WHERE rowid = ?;`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}
