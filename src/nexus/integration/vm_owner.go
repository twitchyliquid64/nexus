package integration

import (
	"nexus/data/user"

	"github.com/robertkrimen/otto"
)

type ownerInfoInitialiser struct{}

func (b *ownerInfoInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("id", r.Base.OwnerID); err != nil {
		return err
	}
	if err := obj.Set("get", func(call otto.FunctionCall) otto.Value {
		usr, _ := user.GetByUID(r.Ctx, r.Base.OwnerID, db)
		result, _ := r.VM.ToValue(usr)
		return result
	}); err != nil {
		return err
	}

	return r.VM.Set("owner", obj)
}
