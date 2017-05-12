package user

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ErrUserDoesntExist is returned when a user does not exist
var ErrUserDoesntExist = errors.New("user does not exist")

// Table (user) implements the databaseTable interface.
type Table struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *Table) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS users (
	  username STRING NOT NULL,
	  display_name STRING,
	  created_at TIME NOT NULL,
	  passhash_if_no_auth_methods BLOB,
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

// Get returns the details of a user
func Get(ctx context.Context, username string, db *sql.DB) (UID int, displayName string, createdAt time.Time, err error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), display_name, created_at FROM users WHERE username = $1;
	`, username)
	if err != nil {
		return -1, "", time.Time{}, err
	}
	defer res.Close()

	if !res.Next() {
		err = ErrUserDoesntExist
		return
	}
	err = res.Scan(&UID, &displayName, &createdAt)
	return
}

// SetAuth sets the default authentication hash for a uid.
func SetAuth(ctx context.Context, uid int, passwd string, db *sql.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(passwd+"yoloSalt"+strconv.Itoa(uid)), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE users SET passhash_if_no_auth_methods=$1 WHERE id() = $2", hash, uid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Create creates a user in the datastore.
func Create(ctx context.Context, username, displayName string, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	INSERT INTO
		users (username, display_name, created_at)
		VALUES (
			$1,
			$2,
			now()
		);
	`, username, displayName)
	if err != nil {
		return err
	}
	return tx.Commit()
}
