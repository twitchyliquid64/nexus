package triggers

import (
	"errors"
	"log"
	"nexus/data/integration"
	"sync"
	"time"

	"github.com/gorhill/cronexpr"
	"github.com/robertkrimen/otto"
)

// CronTriggers holds the state of all the currently extablished CRON triggers.
type CronTriggers struct {
	Start func(runnableUID, triggerID int, runReason string, vm *otto.Otto) (string, error)

	nextCronRun map[int]time.Time      //records the next time a particular triggerID will run.
	triggers    []*integration.Trigger //records all the triggers which represent CRONs
	changeLock  sync.Mutex             //prevent concurrency
}

// Setup is called on system initalisation
func (t *CronTriggers) Setup() {
	t.nextCronRun = map[int]time.Time{}
	go func() {
		for {
			time.Sleep(time.Second * 6)
			t.tickCheck()
		}
	}()
}

func (t *CronTriggers) calcNextRunTime(trigger *integration.Trigger) {
	expr, err := cronexpr.Parse(trigger.Val1)
	if err != nil {
		log.Printf("[CRON][%d] Could not parse timespec - %s", trigger.UID, err.Error())
		return
	}
	t.nextCronRun[trigger.UID] = expr.Next(time.Now())
}

// runs every x seconds to check when a cron should run, and fire them off
func (t *CronTriggers) tickCheck() {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	for _, trigger := range t.triggers {
		if nextRun, known := t.nextCronRun[trigger.UID]; known {
			if nextRun.Before(time.Now()) {
				t.calcNextRunTime(trigger)
				vm := otto.New()
				vm.Set("cronspec", trigger.Val1)
				_, err := t.Start(trigger.ParentUID, trigger.UID, "CRON", vm)
				if err != nil {
					log.Printf("[CRON][%d] Could not start run - %s", trigger.UID, err.Error())
				}
			}
		} else { //not known, set it up
			t.calcNextRunTime(trigger)
		}
	}
}

// New is called when a new cron trigger is registered.
func (t *CronTriggers) New(trigger *integration.Trigger) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	for _, existingTrig := range t.triggers {
		if existingTrig.UID == trigger.UID {
			return errors.New("Trigger already registered")
		}
	}
	t.triggers = append(t.triggers, trigger)
	t.calcNextRunTime(trigger)
	return nil
}

// Delete is called when a cron trigger is removed.
func (t *CronTriggers) Delete(parentRunnableUID int) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	var newTriggerList []*integration.Trigger
	for _, trig := range t.triggers {
		if trig.ParentUID != parentRunnableUID {
			newTriggerList = append(newTriggerList, trig)
		} else {
			if _, exists := t.nextCronRun[trig.UID]; exists {
				delete(t.nextCronRun, trig.UID)
			}
		}
	}
	t.triggers = newTriggerList
	return nil
}
