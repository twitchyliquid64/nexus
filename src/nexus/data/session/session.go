package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"nexus/data/util"
	"nexus/metrics"
	"strconv"
	"time"
)

// ErrInvalidSession is returned when a session does not exist or has been revoked
var ErrInvalidSession = errors.New("invalid session")

// Table (session) implements the databaseTable interface.
type Table struct{}

// AuthKind represents the kind of authentication used to create a session
type AuthKind string

const (
	// AuthPass represents a session created as a result of an authentication with a password.
	AuthPass AuthKind = "PASS"
	// Auth2SC represents a session created as a result of password + softcode authentication.
	Auth2SC AuthKind = "2FASC"
	// Admin represents a session created by an administrator.
	Admin AuthKind = "ADMIN"
)

// Setup is called on initialization to create necessary structures in the database.
func (t *Table) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  uid int NOT NULL,
	  sid varchar(64) NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  revoked BOOLEAN NOT NULL DEFAULT 0,
    can_access_web BOOLEAN NOT NULL DEFAULT 1,
    can_access_sys_api BOOLEAN NOT NULL DEFAULT 0,
    authed_via varchar(64),
		auth_data varchar(4096) NOT NULL DEFAULT "{}"
	);

  CREATE INDEX IF NOT EXISTS sessions_sid ON sessions(sid);
  CREATE INDEX IF NOT EXISTS sessions_uid ON sessions(uid);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return t.migrationDataColumn(ctx, db)
}

func (t *Table) migrationDataColumn(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "SELECT auth_data FROM sessions LIMIT 1;")
	if err == nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE sessions ADD COLUMN auth_data varchar(4096) NOT NULL DEFAULT "{}";`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *Table) Forms() []*util.FormDescriptor {
	return []*util.FormDescriptor{
		&util.FormDescriptor{
			SettingsTitle: "Sessions",
			ID:            "userSessions",
			Desc:          "List of sessions and related settings for this user.",
			Tables: []*util.TableDescriptor{
				&util.TableDescriptor{
					Name: "All Sessions",
					Desc: "All sessions we have on file.",
					ID:   "sessions_list",
					Cols: []string{"#", "Created", "revoked?", "Score", "Methods"},
					FetchContent: func(ctx context.Context, uid int, db *sql.DB) ([]interface{}, error) {
						data, err := GetAllForUser(ctx, uid, db)
						if err != nil {
							return nil, err
						}
						out := make([]interface{}, len(data))
						for i, s := range data {
							authData := map[string]interface{}{}
							json.Unmarshal([]byte(s.AuthDataRaw), &authData)
							out[i] = []interface{}{s.SessionUID, s.Created.Format(time.Stamp), s.Revoked, authData["TotalScore"], authData["PassedMethod"]}
						}
						return out, nil
					},
					Actions: []*util.TableAction{
						&util.TableAction{
							ID:           "sessions_list_revoke",
							Action:       "Revoke",
							MaterialIcon: "block",
							Handler: func(rowID, formID, actionUID string, userID int, db *sql.DB) error {
								uid, err := strconv.Atoi(rowID)
								if err != nil {
									return nil
								}
								ctx := context.Background()
								s, err := GetByUID(ctx, uid, db)
								if err != nil {
									return nil
								}
								if s.UID != userID {
									return errors.New("Session is not owned by requesting user")
								}
								return Revoke(ctx, s.SID, db)
							},
						},
						&util.TableAction{
							ID:           "sessions_list_delete",
							Action:       "Delete",
							MaterialIcon: "delete",
							Handler: func(rowID, formID, actionUID string, userID int, db *sql.DB) error {
								uid, err := strconv.Atoi(rowID)
								if err != nil {
									return nil
								}
								ctx := context.Background()
								s, err := GetByUID(ctx, uid, db)
								if err != nil {
									return nil
								}
								if s.UID != userID {
									return errors.New("Session is not owned by requesting user")
								}
								return Delete(ctx, s.SID, db)
							},
						},
					},
				},
			},
		},
	}
}

// DAO represents a stored session.
type DAO struct {
	SessionUID int
	UID        int
	SID        string
	Created    time.Time

	AccessWeb bool
	AccessAPI bool
	AuthedVia string
	Revoked   bool

	AuthDataRaw string
}

// GetAllForUser is called to get all sessions for a given uid.
func GetAllForUser(ctx context.Context, uid int, db *sql.DB) ([]*DAO, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, sid, created_at, can_access_web, can_access_sys_api, authed_via, revoked, auth_data
		FROM sessions
		WHERE uid = ?
		ORDER BY created_at DESC;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*DAO
	for res.Next() {
		var o DAO
		o.UID = uid
		if err := res.Scan(&o.SessionUID, &o.SID, &o.Created, &o.AccessWeb, &o.AccessAPI, &o.AuthedVia, &o.Revoked, &o.AuthDataRaw); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// GetByUID is called to get the details of a session by UID.
func GetByUID(ctx context.Context, uid int, db *sql.DB) (*DAO, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, uid, created_at, can_access_web, can_access_sys_api, authed_via, sid
		FROM sessions
		WHERE rowid = ?;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrInvalidSession
	}
	var o DAO
	return &o, res.Scan(&o.SessionUID, &o.UID, &o.Created, &o.AccessWeb, &o.AccessAPI, &o.AuthedVia, &o.SID)
}

// Get is called to get the details of a session. Returns an error if the session does not exist or is revoked.
func Get(ctx context.Context, sid string, getDetails bool, db *sql.DB) (*DAO, error) {
	defer metrics.GetSessionSIDDbTime.Time(time.Now())
	query := `SELECT rowid, uid, created_at, can_access_web, can_access_sys_api, authed_via FROM sessions WHERE sid = ? AND revoked = 0;`
	if getDetails {
		query = `SELECT rowid, uid, created_at, can_access_web, can_access_sys_api, authed_via, auth_data FROM sessions WHERE sid = ? AND revoked = 0;`
	}

	res, err := db.QueryContext(ctx, query, sid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrInvalidSession
	}
	var o DAO
	o.SID = sid
	if getDetails {
		return &o, res.Scan(&o.SessionUID, &o.UID, &o.Created, &o.AccessWeb, &o.AccessAPI, &o.AuthedVia, &o.AuthDataRaw)
	}
	return &o, res.Scan(&o.SessionUID, &o.UID, &o.Created, &o.AccessWeb, &o.AccessAPI, &o.AuthedVia)
}

// Create creates a session in the datastore.
func Create(ctx context.Context, uid int, allowWeb, allowAPI bool, authedVia AuthKind, details string, db *sql.DB) (string, error) {
	sid, err := GenerateRandomString(32)
	if err != nil {
		return "", err
	}

	tx, err := db.Begin()
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(`
	INSERT INTO
		sessions (uid, sid, can_access_web, can_access_sys_api, authed_via, auth_data)
		VALUES (?, ?, ?, ?, ?, ?);
	`, uid, sid, allowWeb, allowAPI, string(authedVia), details)
	if err != nil {
		tx.Rollback()
		return "", err
	}
	return sid, tx.Commit()
}

// Revoke sets REVOKE=TRUE for a given session.
func Revoke(ctx context.Context, sid string, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	UPDATE
		sessions SET revoked = 1 WHERE sid = ?;
	`, sid)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// RevokeByAge revokes sessions past a certain age
func RevokeByAge(ctx context.Context, days int, db *sql.DB) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	l, err := tx.ExecContext(ctx, `UPDATE sessions SET revoked = 1 WHERE created_at < ? AND revoked = 0;`, time.Now().AddDate(0, 0, -days))
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	num, err := l.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return num, tx.Commit()
}

// DeleteRevokedByAge deletes revoked sessions past a certain age
func DeleteRevokedByAge(ctx context.Context, days int, db *sql.DB) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	l, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE created_at < ? AND revoked = 1;`, time.Now().AddDate(0, 0, -days))
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	num, err := l.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return num, tx.Commit()
}

// Delete deletes a session with a particular SID
func Delete(ctx context.Context, sid string, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM sessions WHERE sid=?;`, sid)
	if err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
// Sauce: https://elithrar.github.io/article/generating-secure-random-numbers-crypto-rand/
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
// Sauce: https://elithrar.github.io/article/generating-secure-random-numbers-crypto-rand/
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}
