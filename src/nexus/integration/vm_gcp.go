package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"nexus/fs"
	"reflect"

	compute "google.golang.org/api/compute/v1"

	"github.com/robertkrimen/otto"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type gcpInitialiser struct{}

func (b *gcpInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("load_service_credential", func(call otto.FunctionCall) otto.Value {
		gcpCredsPath := call.Argument(0).String()
		var b bytes.Buffer
		err := fs.Contents(r.Ctx, gcpCredsPath, r.Base.OwnerID, &b)
		if err != nil {
			return r.VM.MakeCustomError("fs-error", err.Error())
		}

		conf, err := google.JWTConfigFromJSON(b.Bytes(), compute.CloudPlatformScope)
		if err != nil {
			return r.VM.MakeCustomError("oauth-error", err.Error())
		}
		client := conf.Client(oauth2.NoContext)
		v, err := r.VM.ToValue(client)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		return v
	}); err != nil {
		return err
	}

	if err := obj.Set("compute_client", func(call otto.FunctionCall) otto.Value {
		export, err := call.Argument(0).Export()
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		computeService, err := compute.New(export.(*http.Client))
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		ret, err := makeObject(r.VM)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := b.MakeCompute(r, ret, computeService); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		return ret.Value()
	}); err != nil {
		return err
	}

	if err := obj.Set("pubsub_client", func(call otto.FunctionCall) otto.Value {
		client, err := call.Argument(0).Export()
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		ret, err := makeObject(r.VM)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := b.MakePubSub(r, ret, client.(*http.Client)); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		return ret.Value()
	}); err != nil {
		return err
	}

	return r.VM.Set("gcp", obj)
}

func (b *gcpInitialiser) MakePubSub(r *Run, obj *otto.Object, client *http.Client) error {
	if err := obj.Set("list_topics", func(call otto.FunctionCall) otto.Value {
		resp, err := client.Get("https://pubsub.googleapis.com/v1/projects/" + call.Argument(0).String() + "/topics")
		if err != nil {
			return r.VM.MakeCustomError("pubsub-error", err.Error())
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return r.VM.MakeCustomError("pubsub-error", resp.Status)
		}
		var out map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&out)
		if err != nil {
			return r.VM.MakeCustomError("decode-error", err.Error())
		}
		v, err := r.VM.ToValue(out)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	}); err != nil {
		return err
	}

	if err := obj.Set("create", func(call otto.FunctionCall) otto.Value {
		uri, err := url.Parse("https://pubsub.googleapis.com/v1/projects/" + call.Argument(0).String() + "/topics/" + call.Argument(1).String())
		if err != nil {
			return r.VM.MakeCustomError("format-error", err.Error())
		}
		resp, err := client.Do(&http.Request{Method: "PUT", URL: uri})
		if err != nil {
			return r.VM.MakeCustomError("pubsub-error", err.Error())
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return r.VM.MakeCustomError("pubsub-error", resp.Status)
		}
		var out map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&out)
		if err != nil {
			return r.VM.MakeCustomError("decode-error", err.Error())
		}
		v, err := r.VM.ToValue(out)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	}); err != nil {
		return err
	}

	if err := obj.Set("publish", func(call otto.FunctionCall) otto.Value {
		uri, err := url.Parse("https://pubsub.googleapis.com/v1/projects/" + call.Argument(0).String() + "/topics/" + call.Argument(1).String())
		if err != nil {
			return r.VM.MakeCustomError("format-error", err.Error())
		}

		m, _ := call.Argument(2).Export()
		if _, ok := m.([]map[string]interface{}); !ok {
			return r.VM.MakeCustomError("format-error", "expected a list of pubsub message objects, got "+reflect.TypeOf(m).String())
		}
		jsMessages := m.([]map[string]interface{})
		for i := range jsMessages {
			if _, ok := jsMessages[i]["data"].(string); ok {
				jsMessages[i]["data"] = base64.RawStdEncoding.EncodeToString([]byte(jsMessages[i]["data"].(string)))
			}
		}
		b, err := json.Marshal(map[string]interface{}{"messages": jsMessages})
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		resp, err := client.Post(uri.String()+":publish", "application/json", bytes.NewBuffer(b))
		if err != nil {
			return r.VM.MakeCustomError("pubsub-error", err.Error())
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			e, _ := ioutil.ReadAll(resp.Body)
			return r.VM.MakeCustomError("pubsub-error", string(e))
		}
		var out map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&out)
		if err != nil {
			return r.VM.MakeCustomError("decode-error", err.Error())
		}
		v, err := r.VM.ToValue(out)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	}); err != nil {
		return err
	}

	return nil
}

func (b *gcpInitialiser) MakeCompute(r *Run, obj *otto.Object, service *compute.Service) error {
	return obj.Set("list", func(call otto.FunctionCall) otto.Value {
		instances, err := service.Instances.List(call.Argument(0).String(), call.Argument(1).String()).Do()
		if err != nil {
			return r.VM.MakeCustomError("compute-error", err.Error())
		}
		v, err := r.VM.ToValue(instances.Items)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	})
}
