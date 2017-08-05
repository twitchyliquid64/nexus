package fs

import (
	"context"
	"errors"
	"io"
	"nexus/data/fs"
	"strings"
)

// represents a virtual filesystem.
type source interface {
	Save(ctx context.Context, path string, userID int, data []byte) error
	List(ctx context.Context, path string, userID int) ([]ListResultItem, error)
	Delete(ctx context.Context, p string, userID int) error
	NewFolder(ctx context.Context, p string, userID int) error
	Contents(ctx context.Context, p string, userID int, writer io.Writer) error
	Upload(ctx context.Context, p string, userID int, data io.Reader) error
}

func expandSource(s *fs.Source) (source, error) {
	switch s.Kind {
	case fs.FSSourceMiniFS:
		return &miniFS{}, nil
	case fs.FSSourceS3:
		spl := strings.Split(s.Value1, ":")
		return &s3{
			BucketName: spl[0],
			RegionName: spl[1],
			AccessKey:  s.Value2,
			SecretKey:  s.Value3,
		}, nil
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
