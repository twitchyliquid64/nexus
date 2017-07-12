package fs

import (
	"errors"
	"strings"
)

func validatePath(p string) error {
	if !strings.HasPrefix(p, "/") {
		return errors.New("all paths must lead with '/'")
	}
	if strings.Contains(p, "\n") {
		return errors.New("Invalid characters")
	}
	return nil
}
