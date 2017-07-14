package fs

import (
	"errors"
	"path"
	"strings"
)

func validatePath(p string) (string, error) {
	if !strings.HasPrefix(p, "/") {
		return "", errors.New("all paths must lead with '/'")
	}
	if strings.Contains(p, "\n") {
		return "", errors.New("Invalid characters")
	}
	if len(p) > 1 && p[len(p)-1] == '/' {
		return p[0 : len(p)-2], nil
	}
	return p, nil
}

func computeDir(p string) string {
	if strings.Contains(p, "/") {
		return path.Dir(p)
	}
	return ""
}
