package messaging

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// kinds of message rows
const (
	Msg = "msg"
)

// ErrMessageDoesntExist is returned when the unique ID is not in the database.
var ErrMessageDoesntExist = errors.New("conversation does not exist")

// MessageTable implements the DataTable interface.
type MessageTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *MessageTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS messaging_messages (
    conversation_uid INT NOT NULL,

    content STRING NOT NULL,
	  created_at TIME NOT NULL DEFAULT now(),
	  unique_identifier STRING NOT NULL,
		kind STRING NOT NULL,
    identity STRING
	);

	CREATE UNIQUE INDEX IF NOT EXISTS messaging_messages_uid ON messaging_messages(unique_identifier);
  CREATE INDEX IF NOT EXISTS messaging_messages_conversation ON messaging_messages(conversation_uid);
  CREATE INDEX IF NOT EXISTS messaging_messages_time ON messaging_messages(created_at);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Message is a DAO representing a message.
type Message struct {
	UID int

	Data           string
	Kind           string
	ConversationID int
	UniqueID       string
	From           string

	CreatedAt time.Time
}

// AddMessage records a message against a conversation
func AddMessage(ctx context.Context, msg *Message, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	x, err := tx.Exec(`
	INSERT INTO
		messaging_messages (conversation_uid, content, unique_identifier, kind, identity)
		VALUES ($1, $2, $3, $4, $5);
	`, msg.ConversationID, msg.Data, msg.UniqueID, msg.Kind, msg.From)
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

// GetMessagesForConversation returns a list of messages for a given conversation.
func GetMessagesForConversation(ctx context.Context, convoID int, db *sql.DB) ([]*Message, error) {
	res, err := db.QueryContext(ctx, `
		SELECT id(), kind, content, created_at, unique_identifier, identity FROM messaging_messages
		WHERE conversation_uid = $1 ORDER BY created_at ASC;
	`, convoID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Message
	for res.Next() {
		var o Message
		o.ConversationID = convoID
		if err := res.Scan(&o.UID, &o.Kind, &o.Data, &o.CreatedAt, &o.UniqueID, &o.From); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}

	return output, nil
}
