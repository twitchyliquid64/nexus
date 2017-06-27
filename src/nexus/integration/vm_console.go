package integration

import (
	"fmt"
	"log"
	"strings"

	"nexus/data/integration"

	"github.com/robertkrimen/otto"
)

type consoleInitialiser struct{}

func (b *consoleInitialiser) Apply(r *Run) error {
	val, errMake := r.VM.Get("console")
	if errMake != nil {
		return errMake
	}
	obj := val.Object()

	if err := obj.Set("log", func(call otto.FunctionCall) otto.Value {

		output := []string{}
		for _, argument := range call.ArgumentList {
			output = append(output, fmt.Sprintf("%v", argument))
		}
		outStr := strings.Join(output, " ")

		logErr := integration.WriteLog(r.Ctx, &integration.Log{
			ParentUID: r.Base.UID,
			RunID:     r.ID,
			Value:     outStr,
			Level:     integration.LevelInfo,
			Kind:      integration.KindLog,
		}, db)

		if logErr != nil {
			log.Printf("[run][%s] %q Could not write log line - err: %s", r.ID, r.Base.Name, logErr)
		} else {
			log.Printf("[run][%s][INFO] %s", r.ID, outStr)
		}

		return otto.Value{}
	}); err != nil {
		return err
	}

	return r.VM.Set("console", obj)
}
