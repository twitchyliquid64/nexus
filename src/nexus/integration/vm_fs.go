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

	if err := b.applyRead(obj, r); err != nil {
		return err
	}
	if err := b.applyDelete(obj, r); err != nil {
		return err
	}
	if err := b.applyWrite(obj, r); err != nil {
		return err
	}
	if err := b.applyList(obj, r); err != nil {
		return err
	}
	if err := b.applyHelpers(obj, r); err != nil {
		return err
	}

	return r.VM.Set("fs", obj)
}

func (b *fsInitialiser) applyRead(obj *otto.Object, r *Run) error {
	return obj.Set("read", func(call otto.FunctionCall) otto.Value {
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
	})
}

func (b *fsInitialiser) applyDelete(obj *otto.Object, r *Run) error {
	return obj.Set("delete", func(call otto.FunctionCall) otto.Value {
		path := call.Argument(0).String()
		err := fs.Delete(context.Background(), path, r.Base.OwnerID)
		if err != nil {
			log.Printf("[run][%s][fs.delete] Failed to execute fs.Delete(): %s", r.ID, err)
			return r.VM.MakeCustomError("fs", err.Error())
		}
		return otto.UndefinedValue()
	})
}

func (b *fsInitialiser) applyWrite(obj *otto.Object, r *Run) error {
	return obj.Set("write", func(call otto.FunctionCall) otto.Value {
		path := call.Argument(0).String()
		data := call.Argument(1).String()
		err := fs.Save(context.Background(), path, r.Base.OwnerID, []byte(data))
		if err != nil {
			log.Printf("[run][%s][fs.write] Failed to execute fs.Save(): %s", r.ID, err)
			return r.VM.MakeCustomError("fs", err.Error())
		}
		return otto.UndefinedValue()
	})
}

func (b *fsInitialiser) applyList(obj *otto.Object, r *Run) error {
	return obj.Set("list", func(call otto.FunctionCall) otto.Value {
		path := call.Argument(0).String()

		entries, err := fs.List(context.Background(), path, r.Base.OwnerID)
		if err != nil {
			log.Printf("[run][%s][fs.list] Failed to execute fs.List(): %s", r.ID, err)
			return r.VM.MakeCustomError("fs", err.Error())
		}

		s, err := r.VM.ToValue(entries)
		if err != nil {
			log.Printf("[run][%s][fs.list] Failed to create return value: %s", r.ID, err)
			return r.VM.MakeCustomError("fs-internal", err.Error())
		}

		return s
	})
}

func (b *fsInitialiser) applyHelpers(obj *otto.Object, r *Run) error {
	err := obj.Set("isFile", func(call otto.FunctionCall) otto.Value {
		var kind int64
		if call.Argument(0).IsObject() {
			k, e := call.Argument(0).Object().Get("ItemKind")
			if e != nil {
				return r.VM.MakeCustomError("fs", "isFile() expects a integer or a object from fs.list()")
			}
			kind, _ = k.ToInteger()
		} else {
			kind, _ = call.Argument(0).ToInteger()
		}
		s, _ := otto.ToValue(int(kind) == fs.KindFile)
		return s
	})
	if err != nil {
		return err
	}

	err = obj.Set("isDir", func(call otto.FunctionCall) otto.Value {
		var kind int64
		if call.Argument(0).IsObject() {
			k, e := call.Argument(0).Object().Get("ItemKind")
			if e != nil {
				return r.VM.MakeCustomError("fs", "isDir() expects a integer or a object from fs.list()")
			}
			kind, _ = k.ToInteger()
		} else {
			kind, _ = call.Argument(0).ToInteger()
		}
		s, _ := otto.ToValue(int(kind) == fs.KindDirectory)
		return s
	})

	return err
}
