package integration

import (
	"bytes"
	"context"
	"log"
	"nexus/fs"

	"github.com/robertkrimen/otto"
)

type fsInitialiser struct{}

func (b *fsInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("read", func(call otto.FunctionCall) otto.Value {
		path := call.Argument(0).String()
		var buff bytes.Buffer
		buff.Grow(256)
		err := fs.Contents(context.Background(), path, r.Base.OwnerID, &buff)
		if err != nil {
			log.Printf("[run][%s][fs.read] Failed to execute fs.Contents(): %s", r.ID, err)
			return r.VM.MakeCustomError("fs", err.Error())
		}

		s, _ := otto.ToValue(buff.String())
		return s
	}); err != nil {
		return err
	}

	return r.VM.Set("fs", obj)
}
