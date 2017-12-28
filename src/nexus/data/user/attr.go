package user

import (
	"context"
	"database/sql"
	"nexus/data/dlock"
	"nexus/data/util"
	"time"
)

// attr kinds
const (
	AttrGroup int = iota //attribute represents membership in a group
)

// AttrTable (attr) implements the databaseTable interface.
type AttrTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *AttrTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS user_attr (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  uid INT NOT NULL,
	  kind INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    name VARCHAR(64) NOT NULL,
    val TEXT
	);
	CREATE INDEX IF NOT EXISTS user_attr_uid ON user_attr(uid);
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
func (t *AttrTable) Forms() []*util.FormDescriptor {
	return nil
}

// Attr is the DAO for a user attribute.
type Attr struct {
	UID       int
	UserID    int
	CreatedAt time.Time

	Kind int
	Name string

	Val string
}

// KindStr returns a string representation of the attribute kind.
func (a *Attr) KindStr() string {
	switch a.Kind {
	case AttrGroup:
		return "group"
	default:
		return "?"
	}
}

// GetAttrForUser returns a full list of attributes for the given userID.
func GetAttrForUser(ctx context.Context, UID int, db *sql.DB) ([]*Attr, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `SELECT rowid, uid, kind, created_at, val, name FROM user_attr WHERE uid=?;`, UID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Attr
	for res.Next() {
		var out Attr
		if err := res.Scan(&out.UID, &out.UserID, &out.Kind, &out.CreatedAt, &out.Val, &out.Name); err != nil {
			return nil, err
		}
		output = append(output, &out)
	}
	return output, nil
}

// CreateAttr makes a new attribute.
func CreateAttr(ctx context.Context, attr *Attr, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO
			user_attr (uid, name, kind, val)
			VALUES (?, ?, ?, ?);`, attr.UserID, attr.Name, attr.Kind, attr.Val)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateAttr takes an attribute and updates its Name/kind/Val. Keyed by UID.
func UpdateAttr(ctx context.Context, attr *Attr, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
	UPDATE user_attr SET
		name=?, kind=?, val=? WHERE rowid = ?;`,
		attr.Name, attr.Kind, attr.Val, attr.UID)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// DeleteAttr removes an attribute.
func DeleteAttr(ctx context.Context, id int, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		DELETE FROM
			user_attr WHERE rowid = ?;`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}
