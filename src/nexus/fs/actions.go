package fs

import (
	"context"
	"errors"
	"nexus/data/fs"
	"os"
	"strings"
)

// Action represents a capability present on this file/directory.
type Action struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	ID         string `json:"ID"`
	OutputType string `json:"output_type"`
}

// Action kinds.
const (
	ActionKindButton           = "button"
	ActionKindUnaryStringInput = "1_string"
)

// Output types.
const (
	OutputTypeURL  = "url"
	OutputTypeNone = "none"
)

// Actions returns a list of actions available on a path.
func Actions(ctx context.Context, path string, userID int) ([]Action, error) {
	var err error
	if path, err = validatePath(path); err != nil {
		return nil, err
	}

	if path == "/" { //special case - list actions on sources
		return listActionsOnSources(ctx, path, userID)
	}

	// identify the source and query that
	sources, err := getSourcesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	splitPath := strings.Split(path, "/")
	for _, source := range sources {
		if splitPath[1] == source.Prefix {
			return actionsFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID)
		}
	}
	return nil, os.ErrNotExist
}

func listActionsOnSources(ctx context.Context, path string, userID int) ([]Action, error) {
	return nil, nil
}

func actionsFromSource(ctx context.Context, source *fs.Source, path string, userID int) ([]Action, error) {
	src, err := ExpandSource(source)
	if err != nil {
		return nil, err
	}
	return src.ListActions(ctx, path, userID)
}

func runActionsFromSource(ctx context.Context, source *fs.Source, path string, userID int, id string, payload map[string]string) ([]interface{}, error) {
	src, err := ExpandSource(source)
	if err != nil {
		return nil, err
	}
	return src.RunAction(ctx, path, userID, id, payload)
}

// RunAction runs an action on a path.
func RunAction(ctx context.Context, path string, userID int, id string, payload map[string]string) ([]interface{}, error) {
	var err error
	if path, err = validatePath(path); err != nil {
		return nil, err
	}

	if path == "/" { //special case - list actions on sources
		return nil, errors.New("actions may not be run on root directories")
	}

	// identify the source and query that
	sources, err := getSourcesForUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	splitPath := strings.Split(path, "/")
	for _, source := range sources {
		if splitPath[1] == source.Prefix {
			return runActionsFromSource(ctx, source, strings.Join(splitPath[2:], "/"), userID, id, payload)
		}
	}
	return nil, os.ErrNotExist
}
