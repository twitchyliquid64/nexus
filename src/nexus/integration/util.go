package integration

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"nexus/data/integration"

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

func logControlInfo(ctx context.Context, runID, msg string, runnableUID int, db *sql.DB) error {
	return integration.WriteLog(ctx, &integration.Log{
		ParentUID: runnableUID,
		RunID:     runID,
		Value:     msg,
		Level:     integration.LevelInfo,
		Kind:      integration.KindControlLog,
	}, db)
}

func logControlData(ctx context.Context, runID, msg string, runnableUID, datat int, db *sql.DB) error {
	return integration.WriteLog(ctx, &integration.Log{
		ParentUID: runnableUID,
		RunID:     runID,
		Value:     msg,
		Level:     integration.LevelInfo,
		Kind:      integration.KindStructuredData,
		Datatype:  datat,
	}, db)
}
