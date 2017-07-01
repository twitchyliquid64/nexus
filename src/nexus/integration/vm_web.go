package integration

import (
	"strings"
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
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
	data url.Values
	headers map[string]string
	successCallback, errorCallback otto.Value
}

func ToStringArray(obj otto.Value) ([]string, bool) {
	switch t, _ := obj.Export(); exp := t.(type) {
	case string:
		return []string { exp }, true

	case []interface{}:
		result := make([]string, len(exp))
		for _, val := range exp {
			str, success := val.(string)
			if !success {
				return nil, false
			}

			result = append(result, str)
		}

		return result, true

	default:
		return nil, false
	}
}

func ToHttpValues(vm *otto.Otto, obj *otto.Object) url.Values {
	result := url.Values{}
	for _, key := range obj.Keys() {
		val, err := obj.Get(key)
		if err != nil {
			throwOttoException(vm, "Data object is bad")
		}

		converted, worked := ToStringArray(val)
		if !worked {
			throwOttoException(vm, "Data values must be strings or arrays of strings")
		}

		result[key] = converted
	}

	return result
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
		result.data = ToHttpValues(vm, data.Object())
		callbackIndex += 1
	}

	result.successCallback = call.Argument(callbackIndex)
	if !result.successCallback.IsFunction() {
		throwOttoException(vm, "successCallback must be a function")
	}

	result.errorCallback = call.Argument(callbackIndex + 1)
	if !result.errorCallback.IsFunction() {
		throwOttoException(vm, "errorCallback must be a function")
	}

	return &result
}

func CallError(r *reqArgs, err string) {
	if r.errorCallback.IsFunction() {
		r.errorCallback.Call(otto.NullValue(), err)
	}
}

func createRequest(method, rawUrl string, data url.Values) (*http.Request, error) {
	if data == nil || len(data) == 0 {
		return http.NewRequest(method, rawUrl, nil)
	} else if method == "POST" {
		return http.NewRequest(method, rawUrl, bytes.NewBufferString(data.Encode()))
	} else if method == "GET" {
		req, err := http.NewRequest(method, rawUrl, nil)
		if err != nil {
			return nil, err
		}

		req.URL.RawQuery = data.Encode()
		return req, nil
	} else {
		return nil, errors.New("Unknown request type")
	}
}

func makeWebCall(vm *otto.Otto, method string, details *reqArgs) error {
	req, err := createRequest(method, details.url, details.data)
	for k, v := range details.headers {
		req.Header.Add(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		CallError(details, err.Error())
		return err
	}

	defer resp.Body.Close()
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		CallError(details, readErr.Error())
		return readErr
	}

	details.successCallback.Call(otto.NullValue(), body, resp.StatusCode)
	return nil
}

func addWebCall(vm *otto.Otto, obj *otto.Object, name string) error {
	return obj.Set(name, func(call *otto.FunctionCall) otto.Value {
		makeWebCall(vm, strings.ToUpper(name), determineArgs(vm, call))
		return otto.NullValue()
	})
}

func (b *webInitialiser) Apply(r *Run) error {
	obj, err := makeObject(r.VM);
	if err != nil {
		return err
	}

	if err = addWebCall(r.VM, obj, "get"); err != nil {
		return err
	}

	if err = addWebCall(r.VM, obj, "post"); err != nil {
		return err
	}

	return nil
}
