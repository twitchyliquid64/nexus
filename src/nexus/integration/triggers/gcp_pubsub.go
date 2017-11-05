package triggers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"nexus/data/integration"
	"nexus/fs"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/robertkrimen/otto"
)

type pollResponse struct {
	Msgs []pubsubMessage `json:"receivedMessages"`
}

type pubsubMessage struct {
	AckID   string `json:"ackId"`
	Message struct {
		Data        string            `json:"data"`
		Attributes  map[string]string `json:"attributes"`
		MesssageID  string            `json:"messageId"`
		PublishTime string            `json:"publishTime"`
	} `json:"message"`
}

type subscription struct {
	start func(runnableUID, triggerID int, runReason string, vm *otto.Otto) (string, error)

	name    string
	close   chan bool
	client  *http.Client
	trigger *integration.Trigger
}

func (s *subscription) Close() {
	close(s.close)
}

func (s *subscription) pollRoutine(ctx context.Context, messageChan chan pubsubMessage) {
	project := strings.Split(s.name, "/")[1]
	topic := strings.Split(s.name, "/")[3]
	subName := fmt.Sprintf("projects/%s/subscriptions/nexus-%d-%s", project, s.trigger.ParentUID, topic)

	request, err := http.NewRequest("PUT", "https://pubsub.googleapis.com/v1/"+subName, bytes.NewBufferString("{\"topic\": \""+s.name+"\"}"))
	if err != nil {
		log.Printf("[PUBSUB-INVALID] NewRequest(subscription=%q) err: %v", s.name, err)
		return
	}
	request = request.WithContext(ctx)
	resp, err := s.client.Do(request)
	if err != nil {
		log.Printf("[PUBSUB-POLL] create subscription error: %v", err)
		return
	}
	defer request.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 409 {
		d, _ := ioutil.ReadAll(request.Body)
		log.Printf("[PUBSUB-POLL] %q returned status %q with response body %q", request.URL.String(), resp.Status, string(d))
		log.Printf("[PUBSUB-POLL] Topic name is %q", s.name)
		return
	}

	for {
		reqBody := bytes.NewBufferString("{\"returnImmediately\": false, \"maxMessages\": 5}")
		request, err := http.NewRequest("POST", "https://pubsub.googleapis.com/v1/"+subName+":pull", reqBody)
		if err != nil {
			log.Printf("[PUBSUB-INVALID] NewRequest(subscription=%q) err: %v", s.name, err)
			return
		}
		request = request.WithContext(ctx)
		resp, err := s.client.Do(request)
		if err != nil && strings.Contains(err.Error(), " context canceled") {
			return
		}
		if err != nil {
			log.Printf("[PUBSUB-POLL] poll error: %v", err)
			time.Sleep(time.Second * 8)
			continue
		}
		if resp.StatusCode != 200 {
			log.Printf("[PUBSUB-POLL] had status: %v", resp.Status)
			resp.Body.Close()
			time.Sleep(time.Second * 8)
			continue
		}

		var out pollResponse
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
		if err != nil {
			log.Printf("[PUBSUB-POLL] JSON decode error: %v", err)
			return
		}
		for _, msg := range out.Msgs {
			d, _ := base64.StdEncoding.DecodeString(msg.Message.Data)
			msg.Message.Data = string(d)
			messageChan <- msg
		}
	}
}

func (s *subscription) serviceRoutine() {
	messageChan := make(chan pubsubMessage, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.pollRoutine(ctx, messageChan)

	for {
		select {
		case <-s.close:
			return
		case msg := <-messageChan:
			vm := otto.New()
			requestObj, _ := vm.Object(`pubsub = {}`)
			requestObj.Set("topic_spec", s.name)
			requestObj.Set("trigger_name", s.trigger.Name)
			requestObj.Set("topic", strings.Split(s.name, "/")[3])

			var m map[string]interface{}
			d, _ := json.Marshal(msg.Message)
			json.Unmarshal(d, &m)
			requestObj.Set("message", m)

			requestObj.Set("acknowledge", func(call otto.FunctionCall) otto.Value {
				project := strings.Split(s.name, "/")[1]
				topic := strings.Split(s.name, "/")[3]
				subName := fmt.Sprintf("projects/%s/subscriptions/nexus-%d-%s", project, s.trigger.ParentUID, topic)
				resp, err := s.client.Post("https://pubsub.googleapis.com/v1/"+subName+":acknowledge", "application/json", bytes.NewBufferString("{\"ackIds\":[\""+msg.AckID+"\"]}"))
				if err != nil {
					return vm.MakeCustomError("pubsub-error", err.Error())
				}
				defer resp.Body.Close()
				if resp.StatusCode != 200 {
					return otto.FalseValue()
				}
				return otto.TrueValue()
			})
			vm.Set("pubsub", requestObj)

			_, err := s.start(s.trigger.ParentUID, s.trigger.UID, "PUBSUB", vm)
			if err != nil {
				log.Printf("[PUBSUB][%d] Could not start run - %s", s.trigger.UID, err.Error())
			}
		}
	}
}

// PubsubTriggers holds the state of all the currently extablished pubsub triggers.
type PubsubTriggers struct {
	Start func(runnableUID, triggerID int, runReason string, vm *otto.Otto) (string, error)

	subscriptionsByTrigger map[int]*subscription
	triggers               []*integration.Trigger //records all the triggers which represent pubsub subscriptions
	changeLock             sync.Mutex             //prevent concurrency
}

// Setup is called on system initalisation
func (t *PubsubTriggers) Setup() {
	t.subscriptionsByTrigger = map[int]*subscription{}
}

// New is called when a new pubsub trigger is registered.
func (t *PubsubTriggers) New(trigger *integration.Trigger) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	for _, existingTrig := range t.triggers {
		if existingTrig.UID == trigger.UID {
			return errors.New("Trigger already registered")
		}
	}

	// read credentials file
	var b bytes.Buffer
	err := fs.Contents(context.Background(), trigger.Val2, trigger.OwnerUID, &b)
	if err != nil {
		return err
	}

	conf, err := google.JWTConfigFromJSON(b.Bytes(), "https://www.googleapis.com/auth/pubsub")
	if err != nil {
		return err
	}

	s := &subscription{
		name:    trigger.Val1,
		close:   make(chan bool),
		client:  conf.Client(oauth2.NoContext),
		trigger: trigger,
		start:   t.Start,
	}
	go s.serviceRoutine()
	t.subscriptionsByTrigger[trigger.UID] = s
	t.triggers = append(t.triggers, trigger)
	return nil
}

// Delete is called when a pubsub trigger is removed.
func (t *PubsubTriggers) Delete(parentRunnableUID int) error {
	t.changeLock.Lock()
	defer t.changeLock.Unlock()

	var newTriggerList []*integration.Trigger
	for _, trig := range t.triggers {
		if trig.ParentUID != parentRunnableUID {
			newTriggerList = append(newTriggerList, trig)
		} else {
			t.subscriptionsByTrigger[trig.UID].Close()
			delete(t.subscriptionsByTrigger, trig.UID)
		}
	}
	t.triggers = newTriggerList

	return nil
}
