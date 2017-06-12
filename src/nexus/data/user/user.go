package user

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/datastore"
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
	  created_at TIME NOT NULL DEFAULT now(),
	  passhash_if_no_auth_methods BLOB,

		can_admin_accounts BOOL NOT NULL DEFAULT FALSE,
		can_admin_data BOOL NOT NULL DEFAULT FALSE,
		can_admin_integrations BOOL NOT NULL DEFAULT FALSE,

		is_robot_account BOOL NOT NULL DEFAULT FALSE,
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

// DAO stores information associated with an account.
type DAO struct {
	UID         int
	DisplayName string
	Username    string
	CreatedAt   time.Time

	IsRobot bool

	AdminPerms struct {
		Accounts     bool
		Data         bool
		Integrations bool
	}

	Grants []*datastore.Grant
}

// Update takes the DAO and updates the attributes of the given user. Keyed by UID.
func Update(ctx context.Context, usr *DAO, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
	UPDATE users SET
		username=$2, display_name=$3,
		can_admin_accounts=$4, can_admin_data=$5, can_admin_integrations=$6, is_robot_account=$7

	WHERE id() = $1`, usr.UID, usr.Username, usr.DisplayName, usr.AdminPerms.Accounts, usr.AdminPerms.Data, usr.AdminPerms.Integrations, usr.IsRobot)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// GetByUID looks up the details of an account based on an accounts' UID.
func GetByUID(ctx context.Context, uid int, db *sql.DB) (*DAO, error) {
	res, err := db.QueryContext(ctx, `
		SELECT username, display_name, created_at, can_admin_accounts, can_admin_data, can_admin_integrations, is_robot_account FROM users WHERE id() = $1;
	`, uid)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrUserDoesntExist
	}
	var out DAO
	out.UID = uid
	return &out, res.Scan(&out.Username, &out.DisplayName, &out.CreatedAt, &out.AdminPerms.Accounts, &out.AdminPerms.Data, &out.AdminPerms.Integrations, &out.IsRobot)
}

// Get returns the details of a user
func Get(ctx context.Context, username string, db *sql.DB) (*DAO, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), display_name, created_at, can_admin_accounts, can_admin_data, can_admin_integrations, is_robot_account FROM users WHERE username = $1;
	`, username)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrUserDoesntExist
	}
	var out DAO
	out.Username = username
	return &out, res.Scan(&out.UID, &out.DisplayName, &out.CreatedAt, &out.AdminPerms.Accounts, &out.AdminPerms.Data, &out.AdminPerms.Integrations, &out.IsRobot)
}

// GetAll returns a full list of users in the system.
func GetAll(ctx context.Context, db *sql.DB) ([]*DAO, error) {
	res, err := db.QueryContext(ctx, `SELECT id(), username, display_name, created_at, can_admin_accounts, can_admin_data, can_admin_integrations, is_robot_account FROM users;`)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*DAO
	for res.Next() {
		var out DAO
		if err := res.Scan(&out.UID, &out.Username, &out.DisplayName, &out.CreatedAt, &out.AdminPerms.Accounts, &out.AdminPerms.Data, &out.AdminPerms.Integrations, &out.IsRobot); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}

// CheckBasicAuth returns true if the given password matches the stored hash of the user.
func CheckBasicAuth(ctx context.Context, username, password string, db *sql.DB) (bool, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), passhash_if_no_auth_methods FROM users WHERE username = $1;
	`, username)
	if err != nil {
		return false, err
	}
	defer res.Close()

	if !res.Next() {
		return false, ErrUserDoesntExist
	}

	var uid int
	var hash []byte
	if err = res.Scan(&uid, &hash); err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password+"yoloSalt"+strconv.Itoa(uid))) == nil, nil
}

// SetAuth sets the default authentication hash for a uid.
func SetAuth(ctx context.Context, uid int, passwd string, accAdmin, dataAdmin, integrationAdmin bool, db *sql.DB) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(passwd+"yoloSalt"+strconv.Itoa(uid)), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE users SET passhash_if_no_auth_methods=$1, can_admin_accounts=$3, can_admin_data=$4, can_admin_integrations=$5 WHERE id() = $2", hash, uid, accAdmin, dataAdmin, integrationAdmin)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Delete deletes a user by UID.
func Delete(ctx context.Context, uid int, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM users
			WHERE id() = $1;`, uid)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Create takes the DAO makes a new user with that information.
func Create(ctx context.Context, usr *DAO, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO
			users (username, display_name, created_at, can_admin_accounts, can_admin_data, can_admin_integrations, is_robot_account)
			VALUES (
				$1, $2,
				now(),
				$3, $4, $5, $6
			);`, usr.Username, usr.DisplayName, usr.AdminPerms.Accounts, usr.AdminPerms.Data, usr.AdminPerms.Integrations, usr.IsRobot)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// CreateBasic creates a user in the datastore.
func CreateBasic(ctx context.Context, username, displayName string, db *sql.DB) error {
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
