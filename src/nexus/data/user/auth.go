package user

import (
	"context"
	"database/sql"
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
	  uid INT NOT NULL,
	  kind INT NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),

    class INT NOT NULL,

		val1 STRING,
    val2 STRING,
    val3 STRING,
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

// Auth is a DAO for authentication methods
type Auth struct {
	UID       int
	UserID    int
	CreatedAt time.Time

	Kind  int
	Class int

	Val1, Val2, Val3 string
}

// GetAuthForUser returns a full list of auth methods for the given userID.
func GetAuthForUser(ctx context.Context, UID int, db *sql.DB) ([]*Auth, error) {
	res, err := db.QueryContext(ctx, `SELECT id(), uid, kind, created_at, class, val1, val2, val3 FROM user_auth WHERE uid=$1;`, UID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Auth
	for res.Next() {
		var out Auth
		if err := res.Scan(&out.UID, &out.UserID, &out.Kind, &out.CreatedAt, &out.Class, &out.Val1, &out.Val2, &out.Val3); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}

// GetAuth returns the details of an auth
func GetAuth(ctx context.Context, id int, db *sql.DB) (*Auth, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), uid, kind, created_at, class, val1, val2, val3 FROM user_auth WHERE id() = $1;
	`, id)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrUserDoesntExist
	}
	var out Auth
	return &out, res.Scan(&out.UID, &out.UserID, &out.Kind, &out.CreatedAt, &out.Class, &out.Val1, &out.Val2, &out.Val3)
}

// CreateAuth makes a new auth method.
func CreateAuth(ctx context.Context, auth *Auth, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO
			user_auth (uid, kind, class, val1, val2, val3)
			VALUES ($1, $2, $3, $4, $5, $6);`, auth.UserID, auth.Kind, auth.Class, auth.Val1, auth.Val2, auth.Val3)
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
			user_auth WHERE id() = $1;`, id)
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
		kind=$2, class=$3, val1=$4, val2=$5, val3=$6

	WHERE id() = $1`, auth.UID, auth.Kind, auth.Class, auth.Val1, auth.Val2, auth.Val3)
	if err != nil {
		return err
	}
	return tx.Commit()
}
