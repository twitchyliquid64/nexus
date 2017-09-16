package user

import (
	"context"
	"database/sql"
	"nexus/data/util"
	"time"
)

// auth kinds
const (
	KindOTP int = iota
	KindPassword
)

// auth classes
const (
	ClassRequired int = iota //must always pass for this user
	ClassAccepted            //dont fail if not passed
)

// AuthTable (auth) implements the databaseTable interface.
type AuthTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *AuthTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS user_auth (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  uid INT NOT NULL,
	  kind INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    class INT NOT NULL,
		score INT NOT NULL DEFAULT 1000,

		val1 TEXT,
    val2 TEXT,
    val3 TEXT
	);
	CREATE INDEX IF NOT EXISTS user_auth_uid ON user_auth(uid);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	return t.migrationScoreColumn(ctx, db)
}

func (t *AuthTable) migrationScoreColumn(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "SELECT score FROM user_auth LIMIT 1;")
	if err == nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`ALTER TABLE user_auth ADD COLUMN score INT NOT NULL DEFAULT 1000;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *AuthTable) Forms() []*util.FormDescriptor {
	return nil
}

// Auth is a DAO for authentication methods
type Auth struct {
	UID       int
	UserID    int
	CreatedAt time.Time

	Kind  int
	Class int
	Score int

	Val1, Val2, Val3 string
}

// GetAuthForUser returns a full list of auth methods for the given userID.
func GetAuthForUser(ctx context.Context, UID int, db *sql.DB) ([]*Auth, error) {
	res, err := db.QueryContext(ctx, `SELECT rowid, uid, kind, created_at, class, val1, val2, val3, score FROM user_auth WHERE uid=?;`, UID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Auth
	for res.Next() {
		var out Auth
		if err := res.Scan(&out.UID, &out.UserID, &out.Kind, &out.CreatedAt, &out.Class, &out.Val1, &out.Val2, &out.Val3, &out.Score); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}

// GetAuth returns the details of an auth
func GetAuth(ctx context.Context, id int, db *sql.DB) (*Auth, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, uid, kind, created_at, class, val1, val2, val3, score FROM user_auth WHERE rowid = ?;
	`, id)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrUserDoesntExist
	}
	var out Auth
	return &out, res.Scan(&out.UID, &out.UserID, &out.Kind, &out.CreatedAt, &out.Class, &out.Val1, &out.Val2, &out.Val3, &out.Score)
}

// CreateAuth makes a new auth method.
func CreateAuth(ctx context.Context, auth *Auth, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO
			user_auth (uid, kind, class, val1, val2, val3, score)
			VALUES (?, ?, ?, ?, ?, ?, ?);`, auth.UserID, auth.Kind, auth.Class, auth.Val1, auth.Val2, auth.Val3, auth.Score)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteAuth removes an auth method.
func DeleteAuth(ctx context.Context, id int, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM
			user_auth WHERE rowid = ?;`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateAuth updates an Auth object.
func UpdateAuth(ctx context.Context, auth *Auth, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
	UPDATE user_auth SET
		kind=?, class=?, val1=?, val2=?, val3=?, score=?
			WHERE rowid = ?`, auth.Kind, auth.Class, auth.Val1, auth.Val2, auth.Val3, auth.UID, auth.Score)
	if err != nil {
		return err
	}
	return tx.Commit()
}
