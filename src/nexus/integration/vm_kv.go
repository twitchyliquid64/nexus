package integration

import (
	"context"
	"encoding/json"
	"log"
	"nexus/data/integration"

	"github.com/robertkrimen/otto"
)

type kvInitialiser struct{}

func (b *kvInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("get", func(call otto.FunctionCall) otto.Value {
		row, err := integration.GetStdData(context.Background(), r.Base.UID, call.Argument(0).String(), db)
		if err == integration.ErrNoStdRow {
			return otto.Value{}
		} else if err != nil {
			log.Printf("[run][%s][kv.set] Failed to query: %s", r.ID, err)
			return r.VM.MakeCustomError("kv", err.Error())
		}

		var output interface{}
		err = json.Unmarshal([]byte(row.Value), &output)
		if err != nil {
			log.Printf("[run][%s][kv.set] Failed to json.Unmarshal(): %s", r.ID, err)
			return r.VM.MakeCustomError("kv", err.Error())
		}

		result, _ := r.VM.ToValue(output)
		return result
	}); err != nil {
		return err
	}

	if err := obj.Set("set", func(call otto.FunctionCall) otto.Value {
		export, err := call.Argument(1).Export()
		if err != nil {
			log.Printf("[run][%s][kv.set] Failed to export argument: %s", r.ID, err)
			return r.VM.MakeCustomError("kv", err.Error())
		}

		b, err := json.Marshal(export)
		if err != nil {
			log.Printf("[run][%s][kv.set] Failed to json.Marshal(): %s", r.ID, err)
			return r.VM.MakeCustomError("kv", err.Error())
		}

		err = integration.WriteStdData(context.Background(), r.Base.UID, call.Argument(0).String(), string(b), db)
		if err != nil {
			log.Printf("[run][%s][kv.set] Failed to save: %s", r.ID, err)
			return r.VM.MakeCustomError("kv", err.Error())
		}
		return otto.Value{}
	}); err != nil {
		return err
	}

	return r.VM.Set("kv", obj)
}
