package integration

import (
  "context"
	"log"
  "nexus/data/datastore"

	"github.com/robertkrimen/otto"
)

type datastoreInitialiser struct{}

type functionBinder func(*otto.Object, *Run)error

var binders = []functionBinder{
  bindInsert,
}

func bindInsert(obj *otto.Object, r *Run) error {
  return obj.Set("insert", func(call otto.FunctionCall) otto.Value {
    ctx := context.Background()
    datastoreName := call.Argument(0).String()
    storedDS, err := datastore.GetDatastoreByName(ctx, datastoreName, db)
    if err != nil {
      log.Printf("[run][%s][datastore.insert] Could not read DB by that name: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
    }

    // check allowed to access
    if storedDS.OwnerID != r.Base.OwnerID {
      canAccess, errAccess := datastore.CheckAccess(ctx, r.Base.OwnerID, storedDS.UID, false, db)
      if errAccess != nil {
        return r.VM.MakeCustomError("datastore-internal err", errAccess.Error())
      }
      if !canAccess {
        return r.VM.MakeCustomError("datastore", "cannot insert to datastore you do not own")
      }
    }

    // convert fields to map[string]interface{}
		export, err := call.Argument(1).Export()
		if err != nil {
			log.Printf("[run][%s][datastore.insert] Failed to export argument: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}
    if _, isMap := export.(map[string]interface{}); !isMap{
      return r.VM.MakeCustomError("datastore", "Expected object containing fields to insert")
    }
    fields := export.(map[string]interface{})

    rowID, err := datastore.InsertRow(ctx, storedDS.UID, fields, db)
    if err != nil {
      log.Printf("[run][%s][datastore.insert] Failed to insert: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

    resultObj, err := makeObject(r.VM)
		if err != nil {
			log.Printf("[RUN][%s][datastore.insert] makeObject Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("datastore-internal err", err.Error())
		}
    resultObj.Set("rowID", rowID)
    resultObj.Set("success", true)
		return resultObj.Value()
	})
}

func (b *datastoreInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

  for _, binder := range binders {
    if err := binder(obj, r); err != nil {
      return err
    }
  }

	return r.VM.Set("datastore", obj)
}
