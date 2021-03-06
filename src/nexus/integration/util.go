package integration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"nexus/data/integration"
	notify "nexus/integration/log"

	"github.com/robertkrimen/otto"
)

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
// Sauce: https://elithrar.github.io/article/generating-secure-random-numbers-crypto-rand/
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a URL-safe, base64 encoded
// securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
// Sauce: https://elithrar.github.io/article/generating-secure-random-numbers-crypto-rand/
func GenerateRandomString(s int) (string, error) {
	b, err := GenerateRandomBytes(s)
	return base64.URLEncoding.EncodeToString(b), err
}

func makeObject(o *otto.Otto) (*otto.Object, error) {
	v, err := o.Call("new Object", nil)
	if err != nil {
		return nil, err
	}
	return v.Object(), nil
}

// Panics with an otto Error. This will be caught by otto and converted into a
// JavaScript exception. Only valid inside otto function calls!!
func throwOttoException(vm *otto.Otto, message string) {
	// https://www.bountysource.com/issues/33978990-best-way-to-throw-an-exception-from-go-land
	panic(vm.MakeCustomError("LibException", message))
}

func logControlInfo(ctx context.Context, runID, msg string, runnableUID int, db *sql.DB) error {
	packedMsg := &integration.Log{
		ParentUID: runnableUID,
		RunID:     runID,
		Value:     msg,
		Level:     integration.LevelInfo,
		Kind:      integration.KindControlLog,
	}
	notify.Log(packedMsg)
	return integration.WriteLog(ctx, packedMsg, db)
}

func logControlData(ctx context.Context, runID, msg string, runnableUID, datat int, db *sql.DB) error {
	packedMsg := &integration.Log{
		ParentUID: runnableUID,
		RunID:     runID,
		Value:     msg,
		Level:     integration.LevelInfo,
		Kind:      integration.KindStructuredData,
		Datatype:  datat,
	}
	notify.Log(packedMsg)
	return integration.WriteLog(ctx, packedMsg, db)
}

func logSystemError(ctx context.Context, runID string, err error, runnableUID int, db *sql.DB) error {
	var msg string

	if ottoErr, isOttoError := err.(*otto.Error); isOttoError {
		msg = ottoErr.String()
	} else {
		msg = err.Error()
	}

	packedMsg := &integration.Log{
		ParentUID: runnableUID,
		RunID:     runID,
		Value:     msg,
		Level:     integration.LevelError,
		Kind:      integration.KindStructuredData,
		Datatype:  integration.DatatypeTrace,
	}
	notify.Log(packedMsg)

	return integration.WriteLog(ctx, packedMsg, db)
}
