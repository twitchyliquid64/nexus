package slack

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"nexus/data/messaging"
	"reflect"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

const updateStateDuration = time.Minute * 7

// Source represents a integration with a slack channel for a user.
type Source struct {
	slack *slack.Client
	src   *messaging.Source
	db    *sql.DB

	closeChan    chan bool
	updateTicker *time.Ticker
	wg           *sync.WaitGroup

	channelCache map[string]slack.Channel
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
		channelCache: map[string]slack.Channel{},
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
		log.Printf("ID: %s, Name: %s\n", channel.ID, channel.Name)
		_, err := messaging.GetConversation(ctx, channel.ID, s.src.UID, s.db)
		if err == messaging.ErrConvoDoesntExist {
			err = messaging.AddConversation(ctx, messaging.Conversation{
				SourceUID: s.src.UID,
				UniqueID:  channel.ID,
				Name:      channel.Name,
			}, s.db)
		}
		if err != nil {
			return err
		}
		s.channelCache[channel.ID] = channel
	}
	return nil
}

func (s *Source) getChannelByName(n string) slack.Channel {
	for _, val := range s.channelCache {
		if val.Name == n {
			return val
		}
	}
	return slack.Channel{}
}

// Stop stops listening to events and closes all resources.
func (s *Source) Stop() {
	close(s.closeChan)
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

			case *slack.ConnectedEvent:
				//fmt.Println("Infos:", ev.Info)
				//fmt.Println("Connection counter:", ev.ConnectionCount)
				//rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", s.getChannelByName("general").ID))

			case *slack.MessageEvent:
				fmt.Printf("Message: %v\n", ev)

			case *slack.PresenceChangeEvent:
				fmt.Printf("Presence Change: %v\n", ev)

			case *slack.LatencyReport:
				fmt.Printf("Current latency: %v\n", ev.Value)

			case *slack.RTMError:
				log.Printf("Slack Error: %s\n", ev.Error())

			default:
				log.Printf("Unexpected: %v\n", reflect.TypeOf(msg.Data))
			}
		}
	}
}
