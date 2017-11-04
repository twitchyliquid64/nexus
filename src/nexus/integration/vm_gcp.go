package integration

import (
	"bytes"
	"net/http"
	"nexus/fs"

	compute "google.golang.org/api/compute/v1"
	pubsub "google.golang.org/api/pubsub/v1"

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

		conf, err := google.JWTConfigFromJSON(b.Bytes(), compute.CloudPlatformScope, pubsub.PubsubScope)
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
		export, err := call.Argument(0).Export()
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		pubsubService, err := pubsub.New(export.(*http.Client))
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		ret, err := makeObject(r.VM)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := b.MakePubSub(r, ret, pubsubService); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		return ret.Value()
	}); err != nil {
		return err
	}

	return r.VM.Set("gcp", obj)
}

func (b *gcpInitialiser) MakePubSub(r *Run, obj *otto.Object, service *pubsub.Service) error {
	if err := obj.Set("list", func(call otto.FunctionCall) otto.Value {
		topics, err := service.Projects.Topics.List(call.Argument(0).String()).Do()
		if err != nil {
			return r.VM.MakeCustomError("pubsub-error", err.Error())
		}
		v, err := r.VM.ToValue(topics.Topics)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	}); err != nil {
		return err
	}

	if err := obj.Set("create", func(call otto.FunctionCall) otto.Value {
		topic, err := service.Projects.Topics.Create(call.Argument(0).String(), &pubsub.Topic{}).Do()
		if err != nil {
			return r.VM.MakeCustomError("pubsub-error", err.Error())
		}
		v, err := r.VM.ToValue(topic)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		return v
	}); err != nil {
		return err
	}

	if err := obj.Set("publish", func(call otto.FunctionCall) otto.Value {
		var out []*pubsub.PubsubMessage
		jsMessages, _ := call.Argument(1).Export()
		for _, msg := range jsMessages.([]interface{}) {
			m := msg.(map[string]interface{})
			output := &pubsub.PubsubMessage{
				Data:       m["data"].(string),
				Attributes: make(map[string]string),
			}
			if attrs, ok := m["attrs"].(map[string]interface{}); ok {
				for key, val := range attrs {
					output.Attributes[key] = val.(string)
				}
			}
			out = append(out, output)
		}

		topic, err := service.Projects.Topics.Publish(call.Argument(0).String(), &pubsub.PublishRequest{Messages: out}).Do()
		if err != nil {
			return r.VM.MakeCustomError("pubsub-error", err.Error())
		}
		v, err := r.VM.ToValue(topic)
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
