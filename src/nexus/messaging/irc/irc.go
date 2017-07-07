package irc

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"log"
	"nexus/data/messaging"
	"strconv"
	"strings"
	"sync"
	"time"

	irc "github.com/thoj/go-ircevent"
)

// Source represents a integration with a IRC channel for a user.
type Source struct {
	conn *irc.Connection
	src  *messaging.Source
	db   *sql.DB

	closeChan       chan bool
	wg              *sync.WaitGroup
	sourceToConvoID map[string]int
}

// Make starts talking to IRC and providing messaging integration.
func Make(ctx context.Context, src *messaging.Source, db *sql.DB, wg *sync.WaitGroup) (*Source, error) {
	if src.Details == nil || src.Details["addr"] == "" || src.Details["user"] == "" || src.Details["nick"] == "" || src.Details["channels"] == "" {
		return nil, errors.New("Invalid source: No addr/user/nick/channels")
	}

	out := &Source{
		conn:            irc.IRC(src.Details["nick"], src.Details["user"]),
		src:             src,
		closeChan:       make(chan bool, 1),
		db:              db,
		wg:              wg,
		sourceToConvoID: map[string]int{},
	}

	out.conn.UseTLS = true
	if src.Details["skipTlsVerify"] != "" {
		out.conn.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	if src.Details["debug"] != "" {
		out.conn.Debug = true
	}
	out.conn.AddCallback("001", func(e *irc.Event) {
		for _, channel := range strings.Split(src.Details["channels"], ",") {
			out.conn.Join(channel)
		}
	})

	out.conn.AddCallback("PRIVMSG", out.privMsg)
	out.conn.AddCallback("JOIN", out.join)

	err := out.conn.Connect(src.Details["addr"])
	if err != nil {
		return nil, err
	}

	out.warmCache()
	go out.runLoop()
	return out, nil
}

// Stop stops listening to events and closes all resources.
func (s *Source) Stop() {
	close(s.closeChan)
}

func convoKind(source string) string {
	if strings.HasPrefix(source, "#") {
		return messaging.ChannelConvo
	}
	return messaging.DM
}

func (s *Source) getConvoID(source string) (int, error) {
	if id, known := s.sourceToConvoID[source]; known {
		return id, nil
	}

	conv, err := messaging.GetConversation(context.Background(), source, s.src.UID, s.db)
	if err == messaging.ErrConvoDoesntExist {
		var cID int
		cID, err = messaging.AddConversation(context.Background(), messaging.Conversation{
			SourceUID: s.src.UID,
			UniqueID:  source,
			Name:      source,
			Kind:      convoKind(source),
		}, s.db)
		if err != nil {
			log.Printf("[IRC] Error adding conversation: %v", err)
			return 0, err
		}
		s.sourceToConvoID[source] = cID
		return cID, nil
	} else if err != nil {
		log.Printf("[IRC] Error adding getting convo: %v", err)
		return 0, err
	} else {
		s.sourceToConvoID[source] = conv.UID
		return conv.UID, nil
	}
}

// callback for private message
func (s *Source) join(event *irc.Event) {
	//log.Printf("[IRC][JOIN] %+v", event)
	s.getConvoID(event.Arguments[0])
}

// callback for private message
func (s *Source) privMsg(event *irc.Event) {
	//log.Printf("[IRC][%v][%v] %v", event.Arguments[0], event.Nick, event.Message())
	//event.Message() contains the message
	//event.Nick Contains the sender
	//event.Arguments[0] Contains the channel

	convoKey := event.Arguments[0]
	if !strings.HasPrefix(convoKey, "#") {
		convoKey = event.Nick
	}

	convoID, err := s.getConvoID(convoKey)
	if err != nil { // already logged
		return
	}

	_, err = messaging.AddMessage(context.Background(), &messaging.Message{
		Data:           event.Message(),
		ConversationID: convoID,
		UniqueID:       strconv.Itoa(int(time.Now().UnixNano())),
		Kind:           messaging.Msg,
		From:           event.Nick,
	}, s.db)
	if err != nil {
		log.Printf("[IRC] Error inserting PRIVMSG: %v", err)
	}
}

func (s *Source) warmCache() {
	convos, err := messaging.GetConversationsForSource(context.Background(), s.src.UID, s.db)
	if err != nil {
		log.Printf("[IRC] Failed to warm convo cache: %v", err)
	}
	for _, convo := range convos {
		s.sourceToConvoID[convo.UniqueID] = convo.UID
	}
}

// HandlesConversationID returns true if this source is responsible for the given conversation.
func (s *Source) HandlesConversationID(cID int) bool {
	for _, id := range s.sourceToConvoID {
		if cID == id {
			return true
		}
	}
	return false
}

// Send handles a message from the user to the given conversation.
func (s *Source) Send(cID int, msg string) error {
	for source, id := range s.sourceToConvoID {
		if cID == id {
			s.conn.Privmsg(source, msg)
			_, err := messaging.AddMessage(context.Background(), &messaging.Message{
				Data:           msg,
				ConversationID: cID,
				UniqueID:       strconv.Itoa(int(time.Now().UnixNano())),
				Kind:           messaging.Msg,
				From:           "Me",
			}, s.db)
			if err != nil {
				log.Printf("[IRC] Error inserting PRIVMSG: %v", err)
			}
			return err
		}
	}
	return errors.New("No matching conversation")
}

func (s *Source) runLoop() {
	s.wg.Add(1)
	defer s.wg.Done()

	go s.conn.Loop()

	for {
		select {
		case <-s.closeChan:
			s.conn.Disconnect()
			return
		}
	}
}
