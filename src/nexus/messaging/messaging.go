package messaging

import (
	"context"
	"database/sql"
	"errors"
	"nexus/data/messaging"
	"nexus/messaging/slack"
	"sync"
)

var workingSources []localMessageSource
var wg sync.WaitGroup

type localMessageSource interface {
	Stop()
}

// Init starts the local messaging system - fetching and delivering messages for all non-remote sources.
func Init(ctx context.Context, db *sql.DB) error {
	srcs, err := messaging.GetAllSources(ctx, db)
	if err != nil {
		return err
	}

	for i := range srcs {
		switch srcs[i].Kind {
		case messaging.Slack:
			src, err := slack.Make(ctx, srcs[i], db, &wg)
			if err != nil {
				return err
			}
			workingSources = append(workingSources, src)
		default:
			return errors.New("Unrecognised source kind: " + srcs[i].Kind)
		}
	}

	return nil
}

// Deinit closes all messsaging sources.
func Deinit() {
	for _, source := range workingSources {
		source.Stop()
	}
	wg.Wait()
}
