package messaging

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/util"
	"nexus/metrics"
	"sort"
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
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
	  name varchar(128) NOT NULL,
    source_uid INT NOT NULL,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  unique_identifier varchar(192) NOT NULL,
		kind varchar(32) NOT NULL
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

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *ConversationTable) Forms() []*util.FormDescriptor {
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

	//only populated in GetConversationsForUser
	NumRecentMessages int
	LatestMsgAt time.Time
}


type ByRecentMsg []*Conversation

func (a ByRecentMsg) Len() int           { return len(a) }
func (a ByRecentMsg) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByRecentMsg) Less(i, j int) bool { return a[i].LatestMsgAt.After(a[j].LatestMsgAt) }

// AddConversation creates a new conversation.
func AddConversation(ctx context.Context, c Conversation, db *sql.DB) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	x, err := tx.Exec(`
	INSERT INTO
		messaging_conversation (name, source_uid, unique_identifier, kind)
		VALUES (?, ?, ?, ?);
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
		SELECT rowid, name, source_uid, created_at, unique_identifier, kind FROM messaging_conversation WHERE unique_identifier = ? AND source_uid = ?;
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

// GetConversationByCID returns a convo based on it's CID.
func GetConversationByCID(ctx context.Context, CID int, db *sql.DB) (*Conversation, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, name, source_uid, created_at, unique_identifier, kind FROM messaging_conversation WHERE rowid = ?;
	`, CID)
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

// GetConversationsForUser returns a list of convos for a given user.
func GetConversationsForUser(ctx context.Context, userID int, db *sql.DB) ([]*Conversation, error) {
	defer metrics.GetConvosUIDDbTime.Time(time.Now())
	res, err := db.QueryContext(ctx, `
		SELECT rowid, name, source_uid, created_at, unique_identifier, kind FROM messaging_conversation
		WHERE source_uid IN (SELECT rowid FROM messaging_source WHERE owner_id = ?);
	`, userID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Conversation
	for res.Next() {
		var o Conversation
		if err := res.Scan(&o.UID, &o.Name, &o.SourceUID, &o.CreatedAt, &o.UniqueID, &o.Kind); err != nil {
			return nil, err
		}
		o.LatestMsgAt, o.NumRecentMessages, err = getMessageStatsForConversation(ctx, o.UID, db)
		if err != nil {
			return nil, err
		}
		output = append(output, &o)
	}

	sort.Sort(ByRecentMsg(output))
	return output, nil
}

// GetConversationsForSource returns a list of convos for a given source.
func GetConversationsForSource(ctx context.Context, sourceID int, db *sql.DB) ([]*Conversation, error) {
	res, err := db.QueryContext(ctx, `
		SELECT rowid, name, source_uid, created_at, unique_identifier, kind FROM messaging_conversation
		WHERE source_uid  = ?;
	`, sourceID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []*Conversation
	for res.Next() {
		var o Conversation
		if err := res.Scan(&o.UID, &o.Name, &o.SourceUID, &o.CreatedAt, &o.UniqueID, &o.Kind); err != nil {
			return nil, err
		}
		output = append(output, &o)
	}

	return output, nil
}
