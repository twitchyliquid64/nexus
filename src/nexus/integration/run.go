package integration

import (
	"context"
	"fmt"
	"log"
	"nexus/data/integration"
	"time"

	"github.com/robertkrimen/otto"
)

// register all builtins here
var initialisers = []builtin{
	&basicInfoInitialiser{},
	&ownerInfoInitialiser{},
	&consoleInitialiser{},
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
func Start(runnableUID int, startContext *StartContext) (string, error) {
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
		VM:           otto.New(),
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
	v, runErr := r.VM.Run(r.Base.Content)

	if runErr != nil {
		logSystemError(r.Ctx, r.ID, runErr, r.Base.UID, db)
	}
	logControlData(r.Ctx, r.ID, fmt.Sprintf("value=%v,error='%v'", v, runErr), r.Base.UID, integration.DatatypeEndInfo, db)
	log.Printf("[run][%s] Finished with: %+v and error %v", r.ID, v, runErr)
}