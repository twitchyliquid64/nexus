package integration

import (
	"net/smtp"
	"strings"

	"github.com/robertkrimen/otto"
)

const gmail = "smtp.gmail.com:587"

func sendEmail(addr, to, from, pass, subject, body string) error {
	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: " + subject + "\n\n" +
		body

	return smtp.SendMail(addr,
		smtp.PlainAuth("", from, pass, strings.Split(addr, ":")[0]),
		from, []string{to}, []byte(msg))
}

type emailInitialiser struct{}

func objDiscardError(o otto.Value, err error) otto.Value {
	return o
}

func (b *emailInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("gmail_addr", gmail); err != nil {
		return err
	}

	if err := obj.Set("send", func(call otto.FunctionCall) otto.Value {
		if !call.Argument(2).IsObject() {
			return r.VM.MakeTypeError("Expected object with email info")
		}

		addr := call.Argument(0).String()
		pass := call.Argument(1).String()
		dataObj := call.Argument(2).Object()

		to := objDiscardError(dataObj.Get("to")).String()
		from := objDiscardError(dataObj.Get("from")).String()
		subject := objDiscardError(dataObj.Get("subject")).String()
		body := objDiscardError(dataObj.Get("body")).String()

		sendErr := sendEmail(addr, to, from, pass, subject, body)
		if sendErr != nil {
			return r.VM.MakeCustomError("email", sendErr.Error())
		}
		return otto.Value{}
	}); err != nil {
		return err
	}

	return r.VM.Set("email", obj)
}
