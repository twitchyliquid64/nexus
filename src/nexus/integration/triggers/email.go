package triggers

import (
	"bytes"
	"errors"
	"log"
	"nexus/data/integration"
	"sync"

	"github.com/jhillyerd/enmime"
	"github.com/robertkrimen/otto"
	"github.com/twitchyliquid64/smtpd"
)

// EmailTriggers holds the state of all the currently extablished email triggers.
type EmailTriggers struct {
	Start func(runnableUID, triggerID int, runReason string, vm *otto.Otto) (string, error)

	triggers   []*integration.Trigger //records all the triggers which represent email handlers
	changeLock sync.Mutex             //prevent concurrency
}

// Setup is called on system initalisation
func (t *EmailTriggers) Setup() {
}

// New is called when a new cron trigger is registered.
func (t *EmailTriggers) New(trigger *integration.Trigger) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	for _, existingTrig := range t.triggers {
		if existingTrig.UID == trigger.UID {
			return errors.New("trigger already registered")
		}
	}
	t.triggers = append(t.triggers, trigger)
	return nil
}

// Delete is called when a cron trigger is removed.
func (t *EmailTriggers) Delete(parentRunnableUID int) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	var newTriggerList []*integration.Trigger
	for _, trig := range t.triggers {
		if trig.ParentUID != parentRunnableUID {
			newTriggerList = append(newTriggerList, trig)
		}
	}
	t.triggers = newTriggerList
	return nil
}

// Recipients returns a map of all valid mailboxes.
func (t *EmailTriggers) Recipients() map[string]bool {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()
	out := map[string]bool{}

	for _, t := range t.triggers {
		out[t.Val1] = true
	}
	return out
}

// HandleMail triggers integrations to handle an email.
func (t *EmailTriggers) HandleMail(msg []byte, meta *smtpd.MsgMetadata) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	for _, trigger := range t.triggers {
		for e := meta.Recipients.Front(); e != nil; e = e.Next() {
			rd := e.Value.(smtpd.RecipientDetails)
			if trigger.Val1 == rd.Local {
				env, err := enmime.ReadEnvelope(bytes.NewBuffer(msg))
				if err != nil {
					log.Printf("[EMAIL][%d]Envelope error: %v\n", trigger.UID, err)
					return err
				}

				vm := otto.New()
				messageObj, _ := vm.Object(`message = {}`)
				messageObj.Set("address", rd.Local)
				messageObj.Set("raw", msg)
				messageObj.Set("address", rd.Local)
				messageObj.Set("was_tls", meta.TLS)
				messageObj.Set("from", meta.From)
				messageObj.Set("domain", meta.Domain)
				messageObj.Set("remote", meta.Remote)
				messageObj.Set("text", env.Text)
				messageObj.Set("html", env.HTML)
				messageObj.Set("get_header", env.GetHeader)
				vm.Set("message", messageObj)
				_, err = t.Start(trigger.ParentUID, trigger.UID, "EMAIL", vm)
				if err != nil {
					log.Printf("[EMAIL][%d] Could not start run - %s", trigger.UID, err.Error())
				}
			}
		}
	}
	return nil
}
