package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"nexus/fs"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"

	"github.com/robertkrimen/otto"
)

type gcalInitialiser struct{}

func (c *gcalInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("load_oauth_credentials", func(call otto.FunctionCall) otto.Value {
		clientCredsPath := call.Argument(0).String()
		var b bytes.Buffer
		err := fs.Contents(r.Ctx, clientCredsPath, r.Base.OwnerID, &b)
		if err != nil {
			return r.VM.MakeCustomError("fs-error", err.Error())
		}

		config, err := google.ConfigFromJSON(b.Bytes(), calendar.CalendarReadonlyScope)
		if err != nil {
			return r.VM.MakeCustomError("oauth-error", err.Error())
		}
		cred, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}
		if err := c.makeCredObject(r, cred, config); err != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		return cred.Value()
	}); err != nil {
		return err
	}

	return r.VM.Set("cal", obj)
}

func (c *gcalInitialiser) makeCredObject(r *Run, obj *otto.Object, config *oauth2.Config) error {
	obj.Set("secret", config.ClientSecret)
	obj.Set("id", config.ClientID)

	obj.Set("interactive_oauth_url", func(call otto.FunctionCall) otto.Value {
		authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		v, err := r.VM.ToValue(authURL)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	})

	obj.Set("get_tok_from_interactive_auth_code", func(call otto.FunctionCall) otto.Value {
		tok, err := config.Exchange(oauth2.NoContext, call.Argument(0).String())
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		b, _ := json.Marshal(tok)
		v, err := r.VM.ToValue(string(b))
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	})

	obj.Set("from_token_file", func(call otto.FunctionCall) otto.Value {
		clientCredsPath := call.Argument(0).String()
		var b bytes.Buffer
		if err := fs.Contents(r.Ctx, clientCredsPath, r.Base.OwnerID, &b); err != nil {
			return r.VM.MakeCustomError("fs-error", err.Error())
		}
		t := &oauth2.Token{}
		if err := json.Unmarshal(b.Bytes(), &t); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		cl := config.Client(context.Background(), t)

		client, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}
		if err := c.makeClientObject(r, client, config, cl); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return client.Value()
	})
	return nil
}

func (c *gcalInitialiser) makeClientObject(r *Run, obj *otto.Object, config *oauth2.Config, client *http.Client) error {
	s, err := calendar.New(client)
	if err != nil {
		return err
	}
	obj.Set("calendars", func(call otto.FunctionCall) otto.Value {
		cals, err := s.CalendarList.List().Do()
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		v, err := r.VM.ToValue(cals.Items)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	})
	obj.Set("upcoming_events", func(call otto.FunctionCall) otto.Value {

		events, err := s.Events.List(call.Argument(0).String()).SingleEvents(true).TimeMin(time.Now().Format(time.RFC3339)).OrderBy("startTime").Do()
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		v, err := r.VM.ToValue(events.Items)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	})
	return nil
}
