package integration

import (
	"errors"
	"nexus/data/integration"
	"nexus/integration/triggers"
	"sync"

	"github.com/robertkrimen/otto"
)

type triggerImplementation interface {
	New(*integration.Trigger) error
	Delete(int) error
	Setup()
}

var triggerMapLock sync.Mutex

// WebTrigger is public to expose ServeHTTP, which is used to handle incoming web requests
var WebTrigger = &triggers.WebTriggers{Start: startRunHandler}

// EmailTrigger exposes methods to handle an email.
var EmailTrigger = &triggers.EmailTriggers{Start: startRunHandler}

// trigger kind -> handler mapping
var triggerHandlers = map[string]triggerImplementation{
	"CRON":   &triggers.CronTriggers{Start: startRunHandler},
	"HTTP":   WebTrigger,
	"PUBSUB": &triggers.PubsubTriggers{Start: startRunHandler},
	"EMAIL":  EmailTrigger,
}

// function pointer injected into trigger handlers. Used to kick off a run.
func startRunHandler(runnableUID, triggerID int, triggerKind string, vm *otto.Otto) (string, error) {
	return Start(runnableUID, &StartContext{
		TriggerKind: triggerKind,
		TriggerUID:  triggerID,
	}, vm)
}

func initialiseTrigger(trigger *integration.Trigger) error {
	handler, handlerExists := triggerHandlers[trigger.Kind]
	if !handlerExists {
		return errors.New("No handler for trigger kind " + trigger.Kind)
	}
	handler.New(trigger)
	return nil
}

// RunnableChanged is called when the data model (including triggers) for a runnable is updated.
// It deletes all registered triggers in internal state for that runnable, before establishing triggers.
func RunnableChanged(runnable *integration.Runnable) error {
	triggerMapLock.Lock()
	defer triggerMapLock.Unlock()

	// Delete existing triggers by the runnables ID
	for _, handler := range triggerHandlers {
		handler.Delete(runnable.UID)
	}

	// Add in triggers being saved
	for _, trigger := range runnable.Triggers {
		err := initialiseTrigger(trigger)
		if err != nil {
			return err
		}
	}
	return nil
}
