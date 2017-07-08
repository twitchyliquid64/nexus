package triggers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"nexus/data/integration"
	"nexus/data/session"
	"nexus/data/user"
	"regexp"
	"sync"
	"time"

	"github.com/robertkrimen/otto"
)

const requestTimeout = 25

// WebTriggers implements the trigger handler for HTTP triggers.
// baseTrigger implements generic state tracking for a trigger.
type WebTriggers struct {
	Start func(runnableUID, triggerID int, runReason string, vm *otto.Otto) (string, error)

	triggers   []*integration.Trigger //records all the triggers which represent HTTP handlers
	changeLock sync.Mutex             //prevent concurrency
}

func (t *WebTriggers) findTriggerForRequest(r *http.Request) (*integration.Trigger, error) {
	for _, trig := range t.triggers {
		didMatch, err := regexp.MatchString(trig.Val1, r.URL.Path)
		if err != nil {
			log.Printf("[WEB][%d] Could not parse URL regex - %s", trig.UID, err.Error())
			continue
		}
		if didMatch {
			return trig, nil
		}
	}
	return nil, nil
}

// ServeHTTP is called when a web request is recieved destined for handling by an integration.
func (t *WebTriggers) ServeHTTP(resp http.ResponseWriter, r *http.Request) {
	trig, err := t.findTriggerForRequest(r)
	if err != nil {
		resp.WriteHeader(500)
		resp.Write([]byte("Internal server error"))
		return
	}
	if trig == nil {
		resp.WriteHeader(404)
		resp.Write([]byte("Page not found"))
		return
	}

	vm := otto.New()
	doneChan := make(chan bool)
	vm.Set("request", t.makeRequestVMObj(trig, r, resp, vm, doneChan))

	_, err = t.Start(trig.ParentUID, trig.UID, "HTTP", vm)
	if err != nil {
		log.Printf("[WEB][%d] Could not start run - %s", trig.UID, err.Error())
	}

	select {
	case <-doneChan:
	case <-time.After(time.Second * requestTimeout):
		resp.WriteHeader(502)
		resp.Write([]byte("Timeout"))
	}
}

func (t *WebTriggers) makeRequestVMObj(trig *integration.Trigger, r *http.Request, resp http.ResponseWriter, vm *otto.Otto, doneChan chan bool) *otto.Object {
	requestObj, _ := vm.Object(`request = {}`)
	requestObj.Set("matched_pattern", trig.Val1)
	requestObj.Set("matched_name", trig.Name)
	requestObj.Set("url", r.URL)
	requestObj.Set("user_agent", r.UserAgent())
	requestObj.Set("referer", r.Referer())
	requestObj.Set("remote_addr", r.RemoteAddr)
	requestObj.Set("method", r.Method)
	requestObj.Set("host", r.Host)
	requestObj.Set("uri", r.RequestURI)

	// methods
	requestObj.Set("done", func(call otto.FunctionCall) otto.Value {
		close(doneChan)
		return otto.Value{}
	})
	requestObj.Set("write", func(call otto.FunctionCall) otto.Value {
		resp.Write([]byte(call.Argument(0).String()))
		return otto.Value{}
	})
	requestObj.Set("write_header", func(call otto.FunctionCall) otto.Value {
		i, err := call.Argument(0).ToInteger()
		if err != nil {
			return vm.MakeTypeError("request.write_header takes integer argument")
		}
		resp.WriteHeader(int(i))
		return otto.Value{}
	})
	requestObj.Set("auth", func(call otto.FunctionCall) otto.Value {
		sidCookie, err := r.Cookie("sid")
		if err != nil {
			return otto.Value{}
		}

		session, err := session.Get(context.Background(), sidCookie.Value, db)
		if err != nil {
			return vm.MakeCustomError("web", err.Error())
		}

		usr, err := user.GetByUID(context.Background(), session.UID, db)
		if err != nil {
			return vm.MakeCustomError("web", err.Error())
		}

		o, _ := vm.Object(`authObj = {}`)
		o.Set("session", session)
		o.Set("user", usr)
		o.Set("authenticated", true)

		return o.Value()
	})

	return requestObj
}

// Setup is called on system initalisation
func (t *WebTriggers) Setup() {
}

// New is called when a new HTTP trigger is registered.
func (t *WebTriggers) New(trigger *integration.Trigger) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	for _, existingTrig := range t.triggers {
		if existingTrig.UID == trigger.UID {
			return errors.New("Trigger already registered")
		}
	}
	t.triggers = append(t.triggers, trigger)
	return nil
}

// Delete is called when a HTTP trigger is removed.
func (t *WebTriggers) Delete(parentRunnableUID int) error {
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
