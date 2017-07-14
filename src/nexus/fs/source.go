package fs

import (
	"context"
	"errors"
	"nexus/data/fs"
)

// represents a virtual filesystem.
type source interface {
	Save(ctx context.Context, path string, userID int, data []byte) error
	List(ctx context.Context, path string, userID int) ([]ListResultItem, error)
	Delete(ctx context.Context, p string, userID int) error
	NewFolder(ctx context.Context, p string, userID int) error
}

func expandSource(s *fs.Source) (source, error) {
	switch s.Kind {
	case fs.FSSourceMiniFS:
		return &miniFS{}, nil
	default:
		return nil, errors.New("Cannot expand unrecognised source")
	}
}

func getSourcesForUser(ctx context.Context, userID int) ([]*fs.Source, error) {
	out, err := fs.GetSourcesForUser(ctx, userID, db)
	out = append(out, &fs.Source{
		OwnerID: userID,
		Prefix:  "minifs",
		Kind:    fs.FSSourceMiniFS,
	})
	return out, err
}
