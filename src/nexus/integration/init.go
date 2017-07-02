package integration

import (
	"context"
	"database/sql"
	"nexus/data/integration"
	"sync"
)

var db *sql.DB
var mapLock sync.Mutex
var runs map[string]*Run

// Initialise is called before all other methods to inject handles to dependencies.
func Initialise(ctx context.Context, database *sql.DB) error {
	db = database
	runs = map[string]*Run{}

	for _, triggerHandler := range triggerHandlers {
		triggerHandler.Setup()
	}

	triggers, err := integration.GetAllTriggers(ctx, database)
	if err != nil {
		return err
	}
	for _, t := range triggers {
		err := initialiseTrigger(t)
		if err != nil {
			return err
		}
	}

	return nil
}
