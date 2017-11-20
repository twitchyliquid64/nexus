package integration

import (
	"context"
	"fmt"
	"log"
	"nexus/data/datastore"
	"reflect"
	"strconv"

	"github.com/robertkrimen/otto"
)

type datastoreInitialiser struct{}

type functionBinder func(*otto.Object, *Run) error

var binders = []functionBinder{
	bindInsert,
	bindQuery,
	bindEditRow,
	bindDeleteRow,
}

func bindEditRow(obj *otto.Object, r *Run) error {
	return obj.Set("editRow", func(call otto.FunctionCall) otto.Value {
		ctx := context.Background()
		datastoreName := call.Argument(0).String()
		storedDS, err := datastore.GetDatastoreByName(ctx, datastoreName, db)
		if err != nil {
			log.Printf("[run][%s][datastore.editRow] Could not read DB by that name: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		// check allowed to access
		if storedDS.OwnerID != r.Base.OwnerID {
			canAccess, errAccess := datastore.CheckAccess(ctx, r.Base.OwnerID, storedDS.UID, false, db)
			if errAccess != nil {
				return r.VM.MakeCustomError("datastore-internal err", errAccess.Error())
			}
			if !canAccess {
				return r.VM.MakeCustomError("datastore", "cannot edit row on a datastore you do not own")
			}
		}
		rowID, err := call.Argument(1).ToInteger()
		if err != nil {
			log.Printf("[run][%s][datastore.editRow] Failed to export argument 1: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		// convert fields to map[string]interface{}
		export, err := call.Argument(2).Export()
		if err != nil {
			log.Printf("[run][%s][datastore.editRow] Failed to export argument 2: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}
		if _, isMap := export.(map[string]interface{}); !isMap {
			return r.VM.MakeCustomError("datastore", "Expected object containing fields to set")
		}
		fields := export.(map[string]interface{})

		err = datastore.EditRow(ctx, storedDS.UID, int(rowID), fields, db)
		if err != nil {
			log.Printf("[run][%s][datastore.editRow] Failed to edit: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		resultObj, err := makeObject(r.VM)
		if err != nil {
			log.Printf("[RUN][%s][datastore.editRow] makeObject Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("datastore-internal err", err.Error())
		}
		resultObj.Set("success", true)
		return resultObj.Value()
	})
}

func bindDeleteRow(obj *otto.Object, r *Run) error {
	return obj.Set("deleteRow", func(call otto.FunctionCall) otto.Value {
		ctx := context.Background()
		datastoreName := call.Argument(0).String()
		storedDS, err := datastore.GetDatastoreByName(ctx, datastoreName, db)
		if err != nil {
			log.Printf("[run][%s][datastore.deleteRow] Could not read DB by that name: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		// check allowed to access
		if storedDS.OwnerID != r.Base.OwnerID {
			canAccess, errAccess := datastore.CheckAccess(ctx, r.Base.OwnerID, storedDS.UID, false, db)
			if errAccess != nil {
				return r.VM.MakeCustomError("datastore-internal err", errAccess.Error())
			}
			if !canAccess {
				return r.VM.MakeCustomError("datastore", "cannot delete row on a datastore you do not own")
			}
		}
		rowID, err := call.Argument(1).ToInteger()
		if err != nil {
			log.Printf("[run][%s][datastore.deleteRow] Failed to export argument 1: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		err = datastore.DeleteRow(ctx, storedDS.UID, int(rowID), db)
		if err != nil {
			log.Printf("[run][%s][datastore.deleteRow] Failed to delete: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		resultObj, err := makeObject(r.VM)
		if err != nil {
			log.Printf("[RUN][%s][datastore.deleteRow] makeObject Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("datastore-internal err", err.Error())
		}
		resultObj.Set("success", true)
		return resultObj.Value()
	})
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
		if _, isMap := export.(map[string]interface{}); !isMap {
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

func colIDByName(cols []*datastore.Column, name string) string {
	for _, col := range cols {
		if col.Name == name {
			return strconv.Itoa(col.UID)
		}
	}
	return "???"
}

func bindQuery(obj *otto.Object, r *Run) error {
	return obj.Set("query", func(call otto.FunctionCall) otto.Value {
		ctx := context.Background()
		datastoreName := call.Argument(0).String()
		storedDS, err := datastore.GetDatastoreByName(ctx, datastoreName, db)
		if err != nil {
			log.Printf("[run][%s][datastore.query] Could not read DB by that name: %s", r.ID, err)
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

		cols, err := datastore.GetColumns(ctx, storedDS.UID, db)
		if err != nil {
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		// convert conditions to []interface{}
		var query datastore.Query
		query.UID = storedDS.UID
		export, err := call.Argument(1).Export()
		if err != nil {
			log.Printf("[run][%s][datastore.query] Failed to export argument: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}
		if export != nil {
			if _, isMap := export.([]map[string]interface{}); !isMap {
				return r.VM.MakeCustomError("datastore", "Expected object containing list of conditions, got "+reflect.TypeOf(export).String())
			}
			conditions := export.([]map[string]interface{})

			// construct Query object
			for i, condition := range conditions {
				_, vOk := condition["value"]
				if _, cOk := condition["column"].(string); !cOk || !vOk {
					return r.VM.MakeCustomError("datastore", fmt.Sprintf("bad conditional at index %d: missing or bad type for 'column' or 'value' keys", i))
				}

				var filter datastore.Filter
				filter.Col = colIDByName(cols, condition["column"].(string))
				filter.Type = "literalConstraint"
				filter.Conditional = "=="
				if cond, exists := condition["condition"].(string); exists && cond != "" {
					filter.Conditional = cond
				}
				filter.Val = condition["value"]
				query.Filters = append(query.Filters, filter)
			}
		}

		if call.Argument(2).IsDefined() {
			limit, e1 := call.Argument(2).ToInteger()
			if e1 != nil {
				return r.VM.MakeCustomError("datastore", "Limit is not an integer")
			}
			query.Limit = int(limit)
			offset, e2 := call.Argument(3).ToInteger()
			if e2 != nil {
				return r.VM.MakeCustomError("datastore", "Offset is not an integer")
			}
			query.Offset = int(offset)
		}

		results, err := datastore.DoQuery(ctx, query, db)
		if err != nil {
			log.Printf("[run][%s][datastore.query] Failed to query: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}

		resultObj, err := makeObject(r.VM)
		if err != nil {
			log.Printf("[RUN][%s][datastore.query] makeObject Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("datastore-internal err", err.Error())
		}
		resultObj.Set("results", results)
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
