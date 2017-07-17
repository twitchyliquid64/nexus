package user

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"nexus/data/datastore"
	"nexus/data/util"
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
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  username varchar(128) NOT NULL,
	  display_name varchar(256),
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  passhash_if_no_auth_methods BLOB,

		can_admin_accounts BOOLEAN NOT NULL DEFAULT 0,
		can_admin_data BOOLEAN NOT NULL DEFAULT 0,
		can_admin_integrations BOOLEAN NOT NULL DEFAULT 0,

		is_robot_account BOOLEAN NOT NULL DEFAULT 0
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
func (t *Table) Forms() []*util.FormDescriptor {
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
	log.Printf("Update %+v", usr)
	_, err = tx.ExecContext(ctx, `
	UPDATE users SET
		username=?, display_name=?,
		can_admin_accounts=?, can_admin_data=?, can_admin_integrations=?, is_robot_account=? WHERE rowid = ?;`,
		usr.Username, usr.DisplayName, usr.AdminPerms.Accounts, usr.AdminPerms.Data, usr.AdminPerms.Integrations, usr.IsRobot, usr.UID)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// GetByUID looks up the details of an account based on an accounts' UID.
func GetByUID(ctx context.Context, uid int, db *sql.DB) (*DAO, error) {
	res, err := db.QueryContext(ctx, `
		SELECT username, display_name, created_at, can_admin_accounts, can_admin_data, can_admin_integrations, is_robot_account FROM users WHERE rowid = ?;
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
		SELECT rowid, display_name, created_at, can_admin_accounts, can_admin_data, can_admin_integrations, is_robot_account FROM users WHERE username = ?;
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
	res, err := db.QueryContext(ctx, `SELECT rowid, username, display_name, created_at, can_admin_accounts, can_admin_data, can_admin_integrations, is_robot_account FROM users;`)
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
func CheckBasicAuth(ctx context.Context, username, password string, db *sql.DB) (bool, string, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, passhash_if_no_auth_methods FROM users WHERE username = ?;
	`, username)
	if err != nil {
		return false, "", err
	}
	defer res.Close()

	if !res.Next() {
		return false, "", ErrUserDoesntExist
	}

	var uid int
	var hash []byte
	if err = res.Scan(&uid, &hash); err != nil {
		return false, "", err
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password+"yoloSalt"+strconv.Itoa(uid))) == nil, "BASICPASS", nil
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
	_, err = tx.ExecContext(ctx, "UPDATE users SET passhash_if_no_auth_methods=?, can_admin_accounts=?, can_admin_data=?, can_admin_integrations=? WHERE rowid = ?;",
		hash, accAdmin, dataAdmin, integrationAdmin, uid)
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
			WHERE rowid = ?;`, uid)
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
				?, ?,
				CURRENT_TIMESTAMP,
				?, ?, ?, ?
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
			?,
			?,
			CURRENT_TIMESTAMP
		);
	`, username, displayName)
	if err != nil {
		return err
	}
	return tx.Commit()
}
