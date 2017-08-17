package messaging

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/util"
	"nexus/metrics"
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
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_uid INT NOT NULL,

    content TEXT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  unique_identifier varchar(192) NOT NULL,
		kind varchar(32) NOT NULL,
    identity varchar(64)
	);

	CREATE UNIQUE INDEX IF NOT EXISTS messaging_messages_uid ON messaging_messages(unique_identifier);
  CREATE INDEX IF NOT EXISTS messaging_messages_conversation ON messaging_messages(conversation_uid);
  CREATE INDEX IF NOT EXISTS messaging_messages_combined ON messaging_messages(conversation_uid, created_at);
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
func (t *MessageTable) Forms() []*util.FormDescriptor {
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
	defer metrics.InsertMessageDbTime.Time(time.Now())
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	x, err := tx.Exec(`
	INSERT INTO
		messaging_messages (conversation_uid, content, unique_identifier, kind, identity)
		VALUES (?, ?, ?, ?, ?);
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
	defer metrics.GetMessagesCIDDbTime.Time(time.Now())
	res, err := db.QueryContext(ctx, `
		SELECT rowid, kind, content, created_at, unique_identifier, identity FROM messaging_messages
		WHERE conversation_uid = ? ORDER BY created_at DESC LIMIT 100;
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

// getMessageStatsForConversation returns recent summary information about a conversation.
// The number of messages in the last 15 hours, and the time of the latest message is returned.
func getMessageStatsForConversation(ctx context.Context, convoID int, db *sql.DB) (time.Time, int, error) {
	res, err := db.QueryContext(ctx, `
		SELECT * FROM
		(SELECT count(created_at)
			FROM messaging_messages
			WHERE
				conversation_uid = ? AND
				created_at >= datetime('now', '-15 hours'))
			as num_last_12,
		(SELECT created_at
			FROM messaging_messages
			WHERE
				conversation_uid = ?
			ORDER BY created_at DESC LIMIT 1)
		 	as latest_stamp;

	`, convoID, convoID)
	if err != nil {
		return time.Time{}, 0, err
	}
	defer res.Close()

	if !res.Next() {
		return time.Time{}, 0, nil
	}
	var numberRecentMsgs int
	var lastMessageRecieved time.Time
	return lastMessageRecieved, numberRecentMsgs, res.Scan(&numberRecentMsgs, &lastMessageRecieved)
}
