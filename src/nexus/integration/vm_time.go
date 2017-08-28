package integration

import (
	"reflect"
	"time"

	"github.com/robertkrimen/otto"
)

type tInitialiser struct{}

func (b *tInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("now", func(call otto.FunctionCall) otto.Value {
		result, _ := r.VM.ToValue(time.Now())
		return result
	}); err != nil {
		return err
	}

	if err := obj.Set("unix", func(call otto.FunctionCall) otto.Value {
		i, err := call.Argument(0).ToInteger()
		if err != nil {
			return r.VM.MakeTypeError(err.Error())
		}
		result, _ := r.VM.ToValue(time.Unix(i, 0))
		return result
	}); err != nil {
		return err
	}

	if err := obj.Set("nano", func(call otto.FunctionCall) otto.Value {
		i, err := call.Argument(0).ToInteger()
		if err != nil {
			return r.VM.MakeTypeError(err.Error())
		}
		result, _ := r.VM.ToValue(time.Unix(i/1000, i%1e9))
		return result
	}); err != nil {
		return err
	}

	if err := obj.Set("addDate", func(call otto.FunctionCall) otto.Value {
		t, err := call.Argument(0).Export()
		if err != nil {
			return r.VM.MakeTypeError(err.Error())
		}
		if _, isTime := t.(time.Time); !isTime {
			return r.VM.MakeTypeError("expected time object, got " + reflect.TypeOf(t).String())
		}

		years, _ := call.Argument(1).ToInteger()
		months, _ := call.Argument(2).ToInteger()
		days, _ := call.Argument(3).ToInteger()

		result, _ := r.VM.ToValue(t.(time.Time).AddDate(int(years), int(months), int(days)))
		return result
	}); err != nil {
		return err
	}

	if err := obj.Set("addTime", func(call otto.FunctionCall) otto.Value {
		t, err := call.Argument(0).Export()
		if err != nil {
			return r.VM.MakeTypeError(err.Error())
		}
		if _, isTime := t.(time.Time); !isTime {
			return r.VM.MakeTypeError("expected time object, got " + reflect.TypeOf(t).String())
		}

		hours, _ := call.Argument(1).ToInteger()
		minutes, _ := call.Argument(2).ToInteger()
		seconds, _ := call.Argument(3).ToInteger()

		result, _ := r.VM.ToValue(t.(time.Time).Add(time.Hour*time.Duration(hours) +
			time.Minute*time.Duration(minutes) +
			time.Second*time.Duration(seconds)))
		return result
	}); err != nil {
		return err
	}

	return r.VM.Set("t", obj)
}
