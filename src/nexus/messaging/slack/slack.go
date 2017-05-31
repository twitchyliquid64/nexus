package slack

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"nexus/data/messaging"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

const updateStateDuration = time.Minute * 22

// Source represents a integration with a slack channel for a user.
type Source struct {
	slack *slack.Client
	src   *messaging.Source
	db    *sql.DB

	closeChan    chan bool
	updateTicker *time.Ticker
	wg           *sync.WaitGroup

	channelCache map[string]int
	imCache      map[string]int
}

// Make starts talking to slack and providing messaging integration.
func Make(ctx context.Context, src *messaging.Source, db *sql.DB, wg *sync.WaitGroup) (*Source, error) {
	if src.Details == nil || src.Details["token"] == "" {
		return nil, errors.New("Invalid source: No token")
	}

	out := &Source{
		slack:        slack.New(src.Details["token"]),
		src:          src,
		closeChan:    make(chan bool, 1),
		db:           db,
		channelCache: map[string]int{},
		imCache:      map[string]int{},
		updateTicker: time.NewTicker(updateStateDuration),
		wg:           wg,
	}

	err := out.syncChannels()
	if err != nil {
		return nil, err
	}
	go out.runLoop()
	return out, nil
}

func (s *Source) syncChannels() error {
	ctx := context.Background()

	chans, err := s.slack.GetChannels(false)
	if err != nil {
		return err
	}
	for _, channel := range chans {
		log.Printf("Syncing slack channel - ID: %s, Name: %s\n", channel.ID, channel.Name)
		err = s.checkEnrollChannel(ctx, channel.ID, channel.Name)
		if err != nil {
			return err
		}
	}

	ims, err := s.slack.GetIMChannels()
	if err != nil {
		return err
	}
	for _, im := range ims {
		usr, err := s.slack.GetUserInfo(im.User)
		if err != nil {
			return err
		}
		log.Printf("Syncing slack IMs - ID: %s, Name: %s\n", im.ID, usr.Name)
		err = s.checkEnrollDM(ctx, im.ID, usr.Name+" ("+usr.RealName+")")
		if err != nil {
			return err
		}
	}
	return nil
}

// ensures a database entry exists for the DM session, and there is a cached relation between channelID <-> conversationID
func (s *Source) checkEnrollDM(ctx context.Context, ID, name string) error {
	if _, ok := s.imCache[ID]; ok {
		return nil
	}

	conv, err := messaging.GetConversation(ctx, ID, s.src.UID, s.db)
	if err == messaging.ErrConvoDoesntExist {
		var cID int
		cID, err = messaging.AddConversation(ctx, messaging.Conversation{
			SourceUID: s.src.UID,
			UniqueID:  ID,
			Name:      name,
			Kind:      messaging.DM,
		}, s.db)
		if err != nil {
			return err
		}
		s.imCache[ID] = cID
		return nil
	} else if err != nil {
		return err
	}

	s.imCache[ID] = conv.UID
	return nil
}

// ensures a database entry exists for the Channel session, and there is a cached relation between channelID <-> conversationID
func (s *Source) checkEnrollChannel(ctx context.Context, ID, name string) error {
	if _, ok := s.channelCache[ID]; ok {
		return nil
	}

	conv, err := messaging.GetConversation(ctx, ID, s.src.UID, s.db)
	if err == messaging.ErrConvoDoesntExist {
		var cID int
		cID, err = messaging.AddConversation(ctx, messaging.Conversation{
			SourceUID: s.src.UID,
			UniqueID:  ID,
			Name:      name,
			Kind:      messaging.ChannelConvo,
		}, s.db)
		if err != nil {
			return err
		}
		s.channelCache[ID] = cID
		return nil
	} else if err != nil {
		return err
	}

	s.channelCache[ID] = conv.UID
	return nil
}

// Stop stops listening to events and closes all resources.
func (s *Source) Stop() {
	close(s.closeChan)
}

func (s *Source) onMessage(e *slack.MessageEvent) error {
	mID := e.Channel + "-" + e.Timestamp
	var conversationID int

	if strings.HasPrefix(e.Channel, "C") { //channel
		if _, ok := s.channelCache[e.Channel]; !ok {
			if err := s.syncChannels(); err != nil {
				return err
			}
		}
		conversationID = s.channelCache[e.Channel]
	}

	if strings.HasPrefix(e.Channel, "D") { //direct message
		if _, ok := s.imCache[e.Channel]; !ok {
			if err := s.syncChannels(); err != nil {
				return err
			}
		}
		conversationID = s.imCache[e.Channel]
	}

	_, err := messaging.AddMessage(context.Background(), &messaging.Message{
		Data:           e.Text,
		ConversationID: conversationID,
		UniqueID:       mID,
		Kind:           messaging.Msg,
		From:           e.User,
	}, s.db)
	return err
}

func (s *Source) runLoop() {
	s.wg.Add(1)
	defer s.wg.Done()

	//logger := log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)
	//slack.SetLogger(logger)
	//s.slack.SetDebug(true)
	rtm := s.slack.NewRTM()
	go rtm.ManageConnection()

	for {
		select {
		case <-s.updateTicker.C:
			err := s.syncChannels()
			if err != nil {
				log.Printf("slack syncChannels ERR: %s", err)
			}
		case <-s.closeChan:
			s.updateTicker.Stop()
			rtm.Disconnect()
			return
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.HelloEvent:
			case *slack.ConnectingEvent:
			case *slack.ReconnectUrlEvent:
			case *slack.AckMessage:
			case *slack.LatencyReport:
			case *slack.ConnectedEvent:

			case *slack.MessageEvent:
				err := s.onMessage(ev)
				if err != nil {
					log.Printf("Slack source failed to commit message: %s", err)
				}

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %v\n", ev)

			case *slack.RTMError:
				log.Printf("Slack Error: %s\n", ev.Error())

			default:
				log.Printf("Unexpected: %v\n", reflect.TypeOf(msg.Data))
			}
		}
	}
}