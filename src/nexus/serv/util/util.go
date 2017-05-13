package util

import (
	"log"
	"net/http"
)

// LogIfErr logs a message if err != nil
func LogIfErr(fmt string, err error) {
	if err != nil {
		log.Printf(fmt, err)
	}
}

// InternalHandlerError sanely handles the logging and HTTP response for errors.
// Returns true if request handling should halt (error)
func InternalHandlerError(component string, response http.ResponseWriter, request *http.Request, err error) bool {
	if err != nil {
		log.Printf("%s Error: %s", component, err)
		http.Error(response, "Internal server error", 500)
		return true
	}
	return false
}
