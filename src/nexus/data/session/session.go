package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
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
	  uid int NOT NULL,
	  sid STRING NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
	  revoked BOOL NOT NULL DEFAULT FALSE,
    can_access_web BOOL NOT NULL DEFAULT TRUE,
    can_access_sys_api BOOL NOT NULL DEFAULT FALSE,
    authed_via STRING authed_via IN ("PASS", "2FASC", "2FAHW", "ROBOCREATED", "ADMIN"),
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
	return nil
}

// DAO represents a stored session.
type DAO struct {
	SessionUID int
	UID        int
	SID        string
	Created    time.Time
	AccessWeb  bool
	AccessAPI  bool
	AuthedVia  string
	Revoked    bool
}

// GetAllForUser is called to get all sessions for a given uid.
func GetAllForUser(ctx context.Context, uid int, db *sql.DB) ([]*DAO, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), sid, created_at, can_access_web, can_access_sys_api, authed_via, revoked FROM sessions WHERE uid = $1;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*DAO
	for res.Next() {
		var o DAO
		o.UID = uid
		if err := res.Scan(&o.SessionUID, &o.SID, &o.Created, &o.AccessWeb, &o.AccessAPI, &o.AuthedVia, &o.Revoked); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}
	return output, nil
}

// Get is called to get the details of a session. Returns an error if the session does not exist or is revoked.
func Get(ctx context.Context, sid string, db *sql.DB) (*DAO, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), uid, created_at, can_access_web, can_access_sys_api, authed_via FROM sessions WHERE sid = $1 AND revoked = FALSE;
	`, sid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrInvalidSession
	}
	var o DAO
	o.SID = sid
	return &o, res.Scan(&o.SessionUID, &o.UID, &o.Created, &o.AccessWeb, &o.AccessAPI, &o.AuthedVia)
}

// Create creates a session in the datastore.
func Create(ctx context.Context, uid int, allowWeb, allowAPI bool, authedVia AuthKind, db *sql.DB) (string, error) {
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
		sessions (uid, sid, can_access_web, can_access_sys_api, authed_via)
		VALUES ($1, $2, $3, $4, $5);
	`, uid, sid, allowWeb, allowAPI, string(authedVia))
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
		sessions SET revoked = TRUE WHERE sid = $1;
	`, sid)
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
