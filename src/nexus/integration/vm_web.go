package integration

import (
	"github.com/robertkrimen/otto"
)

type webInitialiser struct{}

type reqType int
const (
	getMethod = iota
	postMethod
)

type reqArgs struct {
	url string
	data map[string]interface{}
	successCallback, errorCallback *otto.Object
}

func determineArgs(vm *otto.Otto, call *otto.FunctionCall) *reqArgs {
	if len(call.ArgumentList) < 2 {
		throwOttoException(vm, "Need atleast url and callback")
	}

	result := reqArgs {}
	if result.url = call.Argument(0).String(); len(result.url) == 0 {
		throwOttoException(vm, "first arg must be the url")
	}

	callbackIndex := 1
	if data := call.Argument(1); data.IsObject() {
		result.data, _ = data.Export() // error on Export() is deprecated
		callbackIndex = 2
	}

	result.successCallback = otto.Argument(callbackIndex)
	if result.successCallback == nil || !result.successCallback.IsFunction() {
		throwOttoException(vm, "successCallback must be a function")
	}

	result.errorCallback = otto.Argument(callbackIndex + 1)
	if result != nil && !result.IsFunction() {
		throwOttoException(vm, "errorCallback must be a function")
	}

	return result
}

func makeWebCall(vm *otto.VM, method reqType, details *reqArgs) error {

}

func addWebCall(vm *otto.VM, obj *otto.Object, name string, typ reqType) {
	if err := obj.Set(name, func(call otto.FunctionCall) otto.Value {
		makeWebCall(vm, typ, determineArgs(vm, call))
	}); err != nil {
		return err
	}
}

func (b *webInitialiser) Apply(r *Run) error {
	val, err := makeObject(r.VM);
	if errMake != nil {
		return errMake
	}

	obj := val.Object()
	if err = addWebCall(vm, obj, "get", getMethod); err != nil {
		return err
	}

	if err = addWebCall(vm, obj, "post", postMethod); err != nil {
		return err
	}
}
