package util

import (
	"log"
	"net/http"
	"strconv"
	"strings"
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

// ExtractTrailingNumFromPath returns the number trailing the last '/' character.
func ExtractTrailingNumFromPath(path string) (int, error) {
	spl := strings.Split(path, "/")
	return strconv.Atoi(spl[len(spl)-1])
}

// ExtractColumnList returns a list of integers which were ',' separated
func ExtractColumnList(cols string) ([]int, error) {
	spl := strings.Split(cols, ",")
	out := make([]int, len(spl))

	for i := range spl {
		var err error
		out[i], err = strconv.Atoi(spl[i])
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
