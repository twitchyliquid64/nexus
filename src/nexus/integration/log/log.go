// Package log dispatches incoming log messages to subscribers.
package log

import (
	"errors"
	"nexus/data/integration"
	"sync"
)

var (
	// ErrNotRunning indicates no run with that runID is currently in progress.
	ErrNotRunning = errors.New("no run with that runID operating")
	stateLock     sync.Mutex
	subscribers   map[string][]consumer //maps runID to consumers
	runningNow    map[string]bool
)

type consumer interface {
	Message(msg *integration.Log)
	Done()
}

func init() {
	stateLock.Lock()
	subscribers = map[string][]consumer{}
	runningNow = map[string]bool{}
	stateLock.Unlock()
}

// Subscribe enrolls a consumer to recieve log messages.
func Subscribe(runID string, c consumer) error {
	stateLock.Lock()
	defer stateLock.Unlock()

	if !runningNow[runID] {
		return ErrNotRunning
	}

	subscribers[runID] = append(subscribers[runID], c)
	return nil
}

// Log is called by the integration runtime to deliver messages to subscribers.
func Log(msg *integration.Log) {
	stateLock.Lock()
	defer stateLock.Unlock()

	for _, c := range subscribers[msg.RunID] {
		c.Message(msg)
	}
}

// Done is called by the integration runtime to indicate a run has concluded.
func Done(runID string) {
	stateLock.Lock()
	defer stateLock.Unlock()

	for _, c := range subscribers[runID] {
		c.Done()
	}

	delete(subscribers, runID)
	delete(runningNow, runID)
}

// Started is called by the integration runtime to indicate a run has started.
func Started(runID string) {
	stateLock.Lock()
	defer stateLock.Unlock()
	runningNow[runID] = true
}
