package integration

import (
	"context"
	"errors"
	"fmt"
	"log"
	"nexus/data/integration"
	"time"

	notify "nexus/integration/log"

	"github.com/robertkrimen/otto"
)

type builtin interface {
	Apply(r *Run) error
}

// register all builtins here
var initialisers = []builtin{
	&basicInfoInitialiser{},
	&ownerInfoInitialiser{},
	&consoleInitialiser{},
	&webInitialiser{},
	&browserInitialiser{},
	&emailInitialiser{},
	&kvInitialiser{},
	&fsInitialiser{},
	&datastoreInitialiser{},
	&tInitialiser{},
	&gcpInitialiser{},
	&gcalInitialiser{},
	&computeInitialiser{},
}

// Run contains the state of a running runnable.
type Run struct {
	ID   string
	Base *integration.Runnable
	Ctx  context.Context

	Started      time.Time
	StartContext *StartContext

	VM *otto.Otto
}

// StartContext represents the cause of a runnable being started.
type StartContext struct {
	TriggerUID  int
	TriggerKind string
}

// Start loads and executes the runnable with the given UID.
func Start(runnableUID int, startContext *StartContext, vm *otto.Otto) (string, error) {
	ctx := context.Background()

	base, err := integration.GetRunnable(ctx, runnableUID, db)
	if err != nil {
		return "", err
	}

	rid, err := GenerateRandomString(8)
	if err != nil {
		return "", err
	}

	r := &Run{
		ID:           rid,
		Ctx:          ctx,
		Base:         base,
		StartContext: startContext,
		Started:      time.Now(),
		VM:           vm,
	}

	for _, initialiser := range initialisers {
		err := initialiser.Apply(r)
		if err != nil {
			return "", err
		}
	}

	mapLock.Lock()
	runs[rid] = r
	mapLock.Unlock()

	go r.start()
	return rid, nil
}

// Start is called to actually run
func (r *Run) start() {
	log.Printf("[run][%s] %q starting", r.ID, r.Base.Name)
	logControlInfo(r.Ctx, r.ID, "Run starting. Cause: "+r.StartContext.TriggerKind, r.Base.UID, db)
	logControlData(r.Ctx, r.ID, "cause="+r.StartContext.TriggerKind, r.Base.UID, integration.DatatypeStartInfo, db) //TODO: Sanitize triggerKind string

	notify.Started(r.ID)
	defer notify.Done(r.ID)
	defer func() {
		if pan := recover(); pan != nil {
			logSystemError(r.Ctx, r.ID, errors.New("Internal Panic! :: "+fmt.Sprint(pan)), r.Base.UID, db)
			log.Printf("[run][%s] Panic'ed!!!! %v", r.ID, pan)

			mapLock.Lock()
			delete(runs, r.ID)
			mapLock.Unlock()
		}
	}()
	v, runErr := r.VM.Run(r.Base.Content)

	if runErr != nil {
		logSystemError(r.Ctx, r.ID, runErr, r.Base.UID, db)
	}
	logControlData(r.Ctx, r.ID, fmt.Sprintf("value=%v,error='%v'", v, runErr), r.Base.UID, integration.DatatypeEndInfo, db)
	log.Printf("[run][%s] Finished with: %+v and error %v", r.ID, v, runErr)

	mapLock.Lock()
	delete(runs, r.ID)
	mapLock.Unlock()
}
