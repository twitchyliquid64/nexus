package messaging

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// different conversation kinds
const (
	ChannelConvo = "chan"
	DM           = "dm"
)

// ErrConvoDoesntExist is returned from GetConversation if the conversation with the given sourceID and unique identifier is not present in the db.
var ErrConvoDoesntExist = errors.New("conversation does not exist")

// ConversationTable implements the DataTable interface.
type ConversationTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *ConversationTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS messaging_conversation (
	  name STRING NOT NULL,
    source_uid INT NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
	  unique_identifier STRING NOT NULL,
		kind STRING NOT NULL,
	);

	CREATE INDEX IF NOT EXISTS messaging_conversation_uid ON messaging_conversation(unique_identifier);
	CREATE INDEX IF NOT EXISTS messaging_conversation_sid ON messaging_conversation(source_uid);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Conversation is a DAO representing a conversation.
type Conversation struct {
	UID       int
	Name      string
	Kind      string
	SourceUID int
	UniqueID  string
	CreatedAt time.Time
}

// AddConversation creates a new conversation.
func AddConversation(ctx context.Context, c Conversation, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	x, err := tx.Exec(`
	INSERT INTO
		messaging_conversation (name, source_uid, unique_identifier, kind)
		VALUES ($1, $2, $3, $4);
	`, c.Name, c.SourceUID, c.UniqueID, c.Kind)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	id, err := x.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return int(id), tx.Commit()
}

// GetConversation returns a convo based on it's unique ID and messaging source ID.
func GetConversation(ctx context.Context, uniqueID string, sourceUID int, db *sql.DB) (*Conversation, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), name, source_uid, created_at, unique_identifier, kind FROM messaging_conversation WHERE unique_identifier = $1 AND source_uid = $2;
	`, uniqueID, sourceUID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	if !res.Next() {
		return nil, ErrConvoDoesntExist
	}

	var o Conversation
	return &o, res.Scan(&o.UID, &o.Name, &o.SourceUID, &o.CreatedAt, &o.UniqueID, &o.Kind)
}
