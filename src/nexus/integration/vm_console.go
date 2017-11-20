package integration

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"nexus/data/integration"
	notify "nexus/integration/log"

	"github.com/robertkrimen/otto"
)

type consoleInitialiser struct{}

func (b *consoleInitialiser) Apply(r *Run) error {
	val, errMake := r.VM.Get("console")
	if errMake != nil {
		return errMake
	}
	obj := val.Object()

	err := b.ApplyBasicLogMethods(r, obj)
	if err != nil {
		return err
	}
	err = b.ApplyDataLogMethods(r, obj)
	if err != nil {
		return err
	}

	return r.VM.Set("console", obj)
}

func (b *consoleInitialiser) ApplyBasicLogMethods(r *Run, obj *otto.Object) error {
	//  ====== method: log  ======
	if err := obj.Set("log", func(call otto.FunctionCall) otto.Value {

		output := []string{}
		for _, argument := range call.ArgumentList {
			output = append(output, fmt.Sprintf("%v", argument))
		}
		outStr := strings.Join(output, " ")

		msg := &integration.Log{
			ParentUID: r.Base.UID,
			RunID:     r.ID,
			Value:     outStr,
			Level:     integration.LevelInfo,
			Kind:      integration.KindLog,
		}
		notify.Log(msg)
		logErr := integration.WriteLog(r.Ctx, msg, db)

		if logErr != nil {
			log.Printf("[run][%s] %q Could not write log line - err: %s", r.ID, r.Base.Name, logErr)
		} else {
			log.Printf("[run][%s][INFO] %s", r.ID, outStr)
		}

		return otto.Value{}
	}); err != nil {
		return err
	}

	//  ====== method: warn  ======
	if err := obj.Set("warn", func(call otto.FunctionCall) otto.Value {

		output := []string{}
		for _, argument := range call.ArgumentList {
			output = append(output, fmt.Sprintf("%v", argument))
		}
		outStr := strings.Join(output, " ")

		msg := &integration.Log{
			ParentUID: r.Base.UID,
			RunID:     r.ID,
			Value:     outStr,
			Level:     integration.LevelWarning,
			Kind:      integration.KindLog,
		}
		notify.Log(msg)
		logErr := integration.WriteLog(r.Ctx, msg, db)

		if logErr != nil {
			log.Printf("[run][%s] %q Could not write log line - err: %s", r.ID, r.Base.Name, logErr)
		} else {
			log.Printf("[run][%s][WARN] %s", r.ID, outStr)
		}

		return otto.Value{}
	}); err != nil {
		return err
	}

	//  ====== method: error  ======
	if err := obj.Set("error", func(call otto.FunctionCall) otto.Value {

		output := []string{}
		for _, argument := range call.ArgumentList {
			output = append(output, fmt.Sprintf("%v", argument))
		}
		outStr := strings.Join(output, " ")

		msg := &integration.Log{
			ParentUID: r.Base.UID,
			RunID:     r.ID,
			Value:     outStr,
			Level:     integration.LevelError,
			Kind:      integration.KindLog,
		}
		notify.Log(msg)
		logErr := integration.WriteLog(r.Ctx, msg, db)

		if logErr != nil {
			log.Printf("[run][%s] %q Could not write log line - err: %s", r.ID, r.Base.Name, logErr)
		} else {
			log.Printf("[run][%s][ERR] %s", r.ID, outStr)
		}

		return otto.Value{}
	}); err != nil {
		return err
	}
	return nil
}

func (b *consoleInitialiser) ApplyDataLogMethods(r *Run, obj *otto.Object) error {
	//  ====== method: data  ======
	if err := obj.Set("data", func(call otto.FunctionCall) otto.Value {

		if len(call.ArgumentList) != 1 {
			return r.VM.MakeCustomError("APIError", "log.Data() method may only be called with a single argument")
		}

		export, err := call.ArgumentList[0].Export()
		if err != nil {
			log.Printf("[run][%s][log.data] Failed to export argument: %s", r.ID, err)
			return otto.Value{}
		}

		b, err := json.Marshal(export)
		if err != nil {
			log.Printf("[run][%s][log.data] Failed to json.Marshal(): %s", r.ID, err)
			return otto.Value{}
		}

		msg := &integration.Log{
			ParentUID: r.Base.UID,
			RunID:     r.ID,
			Value:     string(b),
			Level:     integration.LevelInfo,
			Kind:      integration.KindJSONData,
		}
		notify.Log(msg)
		logErr := integration.WriteLog(r.Ctx, msg, db)

		if logErr != nil {
			log.Printf("[run][%s] %q Could not write log line - err: %s", r.ID, r.Base.Name, logErr)
		} else {
			log.Printf("[run][%s][DATA] %s", r.ID, string(b))
		}

		return otto.Value{}
	}); err != nil {
		return err
	}

	return nil
}
