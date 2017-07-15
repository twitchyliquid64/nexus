package integration

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/robertkrimen/otto"
)

type webInitialiser struct{}

func (b *webInitialiser) Apply(r *Run) error {
	obj, err := makeObject(r.VM)
	if err != nil {
		return err
	}

	if err := b.bindGet(obj, r); err != nil {
		return err
	}
	if err := b.bindPost(obj, r); err != nil {
		return err
	}
	if err := b.bindValues(obj, r); err != nil {
		return err
	}

	return r.VM.Set("web", obj)
}

func applyRequestParams(obj otto.Value, request *http.Request, client *http.Client) error {
	//tr := &http.Transport{}
	if obj.IsObject() {
		o := obj.Object()
		for _, key := range o.Keys() {
			switch key {
			case "data":
				fallthrough
			case "body":
				d, _ := o.Get(key)
				//request.Header.Add("Content-Length", strconv.Itoa(len(d.String())))
				request.ContentLength = int64(len(d.String()))
				request.Body = ioutil.NopCloser(strings.NewReader(d.String()))
			case "content_type":
				fallthrough
			case "Content-Type":
				fallthrough
			case "contentType":
				ct, _ := o.Get(key)
				request.Header.Add("Content-Type", ct.String())
			case "headers":
				headers, err := o.Get("headers")
				if err != nil {
					return err
				}
				for _, headerName := range headers.Object().Keys() {
					headerData, _ := headers.Object().Get(headerName)
					request.Header.Add(headerName, headerData.String())
				}
			}
		}
	}
	//client.Transport = tr
	return nil
}

func (b *webInitialiser) bindGet(obj *otto.Object, r *Run) error {
	err := obj.Set("get", func(call otto.FunctionCall) otto.Value {

		client := &http.Client{}
		req, err := http.NewRequest("GET", call.Argument(0).String(), nil)
		if err != nil {
			log.Printf("[RUN][%s][web.get] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}

		err = applyRequestParams(call.Argument(1), req, client)
		if err != nil {
			log.Printf("[RUN][%s][web.get] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[RUN][%s][web.get] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[RUN][%s][web.get] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}

		result, _ := r.VM.ToValue(struct {
			Code    int
			CodeStr string
			Data    string
			URL     string
			Cookies []*http.Cookie
			Header  http.Header
		}{
			Data:    string(body),
			URL:     call.Argument(0).String(),
			Code:    resp.StatusCode,
			CodeStr: resp.Status,
			Cookies: resp.Cookies(),
			Header:  resp.Header,
		})
		return result
	})
	return err
}

func (b *webInitialiser) bindPost(obj *otto.Object, r *Run) error {
	err := obj.Set("post", func(call otto.FunctionCall) otto.Value {

		client := &http.Client{}
		req, err := http.NewRequest("POST", call.Argument(0).String(), nil)
		if err != nil {
			log.Printf("[RUN][%s][web.post] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}

		err = applyRequestParams(call.Argument(1), req, client)
		if err != nil {
			log.Printf("[RUN][%s][web.post] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[RUN][%s][web.post] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[RUN][%s][web.post] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("web", err.Error())
		}

		result, _ := r.VM.ToValue(struct {
			Code    int
			CodeStr string
			Data    string
			URL     string
			Cookies []*http.Cookie
			Header  http.Header
		}{
			Data:    string(body),
			URL:     call.Argument(0).String(),
			Code:    resp.StatusCode,
			CodeStr: resp.Status,
			Cookies: resp.Cookies(),
			Header:  resp.Header,
		})
		return result
	})
	return err
}

func (b *webInitialiser) bindValues(obj *otto.Object, r *Run) error {
	err := obj.Set("values", func(call otto.FunctionCall) otto.Value {
		out := url.Values{}

		obj := call.Argument(0)
		if obj.IsObject() {
			o := obj.Object()
			for _, key := range o.Keys() {
				d, _ := o.Get(key)
				out.Set(key, d.String())
			}
		}

		result, _ := r.VM.ToValue(out.Encode())
		return result
	})
	return err
}
