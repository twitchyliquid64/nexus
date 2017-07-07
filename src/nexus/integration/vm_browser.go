package integration

import (
	"bytes"
	"log"

	"github.com/headzoo/surf"
	"github.com/headzoo/surf/agent"
	"github.com/headzoo/surf/browser"
	"github.com/robertkrimen/otto"
)

type browserInitialiser struct{}

func (b *browserInitialiser) Apply(r *Run) error {
	return r.VM.Set("browser", func(call otto.FunctionCall) otto.Value {
		bow := surf.NewBrowser()
		obj, err := makeObject(r.VM)
		if err != nil {
			log.Printf("[RUN][%s][BROWSER] Err: %s", r.ID, err.Error())
			return r.VM.MakeCustomError("browser-internal err", err.Error())
		}

		binders := []func(*browser.Browser, *Run, *otto.Object) error{
			bindOpen,
			bindFind,
			bindTitle,
			bindUserAgents,
			bindBody,
			bindBodyRaw,
			bindCookies,
			bindForm,
		}

		for _, bindMethod := range binders {
			err := bindMethod(bow, r, obj)
			if err != nil {
				log.Printf("[RUN][%s][BROWSER] Bind Err: %s", r.ID, err.Error())
				return r.VM.MakeCustomError("browser-internal err", err.Error())
			}
		}

		return obj.Value()
	})
}

func bindOpen(bow *browser.Browser, r *Run, obj *otto.Object) error {
	err := obj.Set("open", func(call otto.FunctionCall) otto.Value {
		url, err := call.Argument(0).ToString()
		if err != nil {
			return r.VM.MakeTypeError(err.Error())
		}
		err = bow.Open(url)
		if err != nil {
			return r.VM.MakeCustomError("browser err", err.Error())
		}
		return otto.Value{}
	})
	return err
}

func bindTitle(bow *browser.Browser, r *Run, obj *otto.Object) error {
	err := obj.Set("title", func(call otto.FunctionCall) otto.Value {
		ret, _ := otto.ToValue(bow.Title())
		return ret
	})
	return err
}

func bindBody(bow *browser.Browser, r *Run, obj *otto.Object) error {
	err := obj.Set("body", func(call otto.FunctionCall) otto.Value {
		ret, _ := otto.ToValue(bow.Body())
		return ret
	})
	return err
}

func bindBodyRaw(bow *browser.Browser, r *Run, obj *otto.Object) error {
	err := obj.Set("bodyRaw", func(call otto.FunctionCall) otto.Value {
		buf := new(bytes.Buffer)
		bow.Download(buf)
		ret, _ := otto.ToValue(buf.String())
		return ret
	})
	return err
}

func bindCookies(bow *browser.Browser, r *Run, obj *otto.Object) error {
	err := obj.Set("cookies", func(call otto.FunctionCall) otto.Value {
		c := bow.SiteCookies()
		ret, e := r.VM.ToValue(c)
		if e != nil {
			return r.VM.MakeCustomError("browser err", e.Error())
		}
		return ret
	})
	return err
}

func bindUserAgents(bow *browser.Browser, r *Run, obj *otto.Object) error {
	if err := obj.Set("setUserAgent", func(call otto.FunctionCall) otto.Value {
		bow.SetUserAgent(call.Argument(0).String())
		return otto.Value{}
	}); err != nil {
		return err
	}
	if err := obj.Set("setChromeAgent", func(call otto.FunctionCall) otto.Value {
		bow.SetUserAgent(agent.Chrome())
		return otto.Value{}
	}); err != nil {
		return err
	}
	err := obj.Set("setFirefoxAgent", func(call otto.FunctionCall) otto.Value {
		bow.SetUserAgent(agent.Firefox())
		return otto.Value{}
	})
	return err
}

func bindForm(bow *browser.Browser, r *Run, obj *otto.Object) error {
	err := obj.Set("form", func(call otto.FunctionCall) otto.Value {
		form, err := bow.Form(call.Argument(0).String())
		if err != nil {
			return r.VM.MakeCustomError("browser err", err.Error())
		}

		obj, err := makeObject(r.VM)
		if err != nil {
			return r.VM.MakeCustomError("browser err", err.Error())
		}

		obj.Set("set", func(in otto.FunctionCall) otto.Value {
			e := form.Input(in.Argument(0).String(), in.Argument(1).String())
			if e != nil {
				return r.VM.MakeCustomError("browser err", e.Error())
			}
			return otto.TrueValue()
		})

		obj.Set("submit", func(in otto.FunctionCall) otto.Value {
			e := form.Submit()
			if e != nil {
				return r.VM.MakeCustomError("browser err", e.Error())
			}
			return otto.TrueValue()
		})
		return obj.Value()
	})
	return err
}

func bindFind(bow *browser.Browser, r *Run, obj *otto.Object) error {
	err := obj.Set("find", func(call otto.FunctionCall) otto.Value {
		q, err := call.Argument(0).ToString()
		if err != nil {
			return r.VM.MakeTypeError(err.Error())
		}
		expr := bow.Find(q)

		obj, err := makeObject(r.VM)
		if err != nil {
			return r.VM.MakeCustomError("browser err", err.Error())
		}

		obj.Set("text", func(in otto.FunctionCall) otto.Value {
			ret, e := otto.ToValue(expr.Text())
			if e != nil {
				return r.VM.MakeCustomError("browser err", err.Error())
			}
			return ret
		})
		obj.Set("html", func(in otto.FunctionCall) otto.Value {
			content, e := expr.Html()
			if e != nil {
				return r.VM.MakeCustomError("browser err", err.Error())
			}
			ret, e := otto.ToValue(content)
			if e != nil {
				return r.VM.MakeCustomError("browser err", err.Error())
			}
			return ret
		})

		return obj.Value()
	})
	return err
}
