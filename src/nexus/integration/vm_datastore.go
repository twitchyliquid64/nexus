package integration

import (
  "context"
	"log"
  "nexus/data/datastore"
  "reflect"
  "strconv"

	"github.com/robertkrimen/otto"
)

type datastoreInitialiser struct{}

type functionBinder func(*otto.Object, *Run)error

var binders = []functionBinder{
  bindInsert,
  bindQuery,
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

func colIDByName(cols []*datastore.Column, name string) string {
  for _, col := range cols {
    if col.Name == name {
      return strconv.Itoa(col.UID)
    }
  }
  return "???"
}

func coerceType(cols []*datastore.Column, name string, val interface{}) string {
  s, isStr := val.(string)
  if isStr {
    return s
  }

  var col *datastore.Column
  for _, c := range cols {
    if c.Name == name {
      col = c
      break
    }
  }
  if col == nil {
    return "??"
  }

  switch col.Datatype {
  case datastore.STR:
    switch v := val.(type) {
    case int64:
      return strconv.Itoa(int(v))
    }
  case datastore.TIME:
    switch v := val.(type) {
    case int64:
      return strconv.Itoa(int(v) / 1000)
    case float64:
      return strconv.Itoa(int(v) / 1000)
    }
  }
  return "?"
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
		export, err := call.Argument(1).Export()
		if err != nil {
			log.Printf("[run][%s][datastore.query] Failed to export argument: %s", r.ID, err)
			return r.VM.MakeCustomError("datastore", err.Error())
		}
    if _, isMap := export.([]map[string]interface{}); !isMap{
      return r.VM.MakeCustomError("datastore", "Expected object containing list of conditions, got " + reflect.TypeOf(export).String())
    }
    conditions := export.([]map[string]interface{})

    // construct Query object
    var query datastore.Query
    query.UID = storedDS.UID
    for _, condition := range conditions {
      var filter datastore.Filter
      filter.Col = colIDByName(cols, condition["column"].(string))
      filter.Type = "literalConstraint"
      filter.Conditional = "=="
      if cond, exists := condition["condition"].(string); exists && cond != "" {
        filter.Conditional = cond
      }
      filter.Val = coerceType(cols, condition["column"].(string), condition["value"])
      query.Filters = append(query.Filters, filter)
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
