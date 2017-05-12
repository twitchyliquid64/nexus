package util

import "log"

// LogIfErr logs a message if err != nil
func LogIfErr(fmt string, err error) {
	if err != nil {
		log.Printf(fmt, err)
	}
}
