package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/mitchellh/goamz/aws"
	s3lib "github.com/mitchellh/goamz/s3"
	"github.com/rlmcpherson/s3gof3r"
)

type S3 struct {
	SecretKey  string
	AccessKey  string
	BucketName string
	RegionName string
}

func (s *S3) getGoAMZ() *s3lib.Bucket {
	s3cli := s3lib.New(aws.Auth{AccessKey: s.AccessKey, SecretKey: s.SecretKey, Token: ""}, aws.Regions[s.RegionName])
	return s3cli.Bucket(s.BucketName)
}

func (s *S3) getStreamingCredentials() *s3gof3r.S3 {
	keys := s3gof3r.Keys{AccessKey: s.AccessKey, SecretKey: s.SecretKey}
	return s3gof3r.New(strings.Replace(aws.Regions[s.RegionName].S3Endpoint, "https://", "", -1), keys)
}

func (s *S3) getStreamParameters() *s3gof3r.Config {
	return &s3gof3r.Config{
		Concurrency: 2,
		PartSize:    6 * 1024 * 1024,
		NTry:        10,
		Md5Check:    true,
		Scheme:      "https",
		Client:      s3gof3r.ClientWithTimeout(12 * time.Second),
	}
}

// Contents implements Source.
func (s *S3) Contents(ctx context.Context, p string, userID int, writer io.Writer) error {
	s3Reader, _, err := s.getStreamingCredentials().Bucket(s.BucketName).GetReader(p, s.getStreamParameters())
	if err != nil {
		return err
	}
	defer s3Reader.Close()
	_, err = io.Copy(writer, s3Reader)
	return err
}

// Save implements Source.
func (s *S3) Save(ctx context.Context, p string, userID int, data []byte) error {
	sizeToDetect := len(data)
	if sizeToDetect > 1024 {
		sizeToDetect = 1024
	}
	contType := http.DetectContentType(data[:sizeToDetect])

	if len(data) == 0 {
		data = make([]byte, 2)
	}
	return s.getGoAMZ().Put(p, data, contType, s3lib.Private)
}

// Delete implements Source.
func (s *S3) Delete(ctx context.Context, p string, userID int) error {
	return s.getGoAMZ().Del(p)
}

// List implements Source.
func (s *S3) List(ctx context.Context, p string, userID int) ([]ListResultItem, error) {
	if p != "" && !strings.HasSuffix(p, "/") {
		p = p + "/"
	}

	listResp, err := s.getGoAMZ().List(p, "/", "", 3000)
	if err != nil {
		return nil, err
	}

	var out []ListResultItem
	for _, line := range listResp.Contents {
		//fmt.Printf("%+v\n", line)
		if line.Key == p {
			continue
		}
		t, err := time.Parse(time.RFC3339, line.LastModified)
		if err != nil {
			fmt.Println(err)
		}
		out = append(out, ListResultItem{
			Name:      line.Key,
			ItemKind:  KindFile,
			Modified:  t,
			SizeBytes: line.Size,
		})
	}
	for _, commonPrefix := range listResp.CommonPrefixes {
		out = append(out, ListResultItem{
			Name:     path.Join(p, commonPrefix[len(p):]),
			ItemKind: KindDirectory,
		})
	}

	return out, nil
}

// NewFolder implements Source.
func (s *S3) NewFolder(ctx context.Context, p string, userID int) error {
	if strings.HasSuffix(p, "/") {
		return errors.New("unexpected trailing slash")
	}
	return s.Save(ctx, p+"/", userID, []byte(""))
}

// Upload implements Source.
func (s *S3) Upload(ctx context.Context, p string, userID int, data io.Reader) error {
	s3Writer, err := s.getStreamingCredentials().Bucket(s.BucketName).PutWriter(p, nil, nil)
	if err != nil {
		return err
	}
	defer s3Writer.Close()
	_, err = io.Copy(s3Writer, data)
	return err
}

// SignedURL returns a temporary URL through which the file can be accessed anonymously.
func (s *S3) SignedURL(ctx context.Context, p string, expires time.Time, userID int) string {
	return s.getGoAMZ().SignedURL(p, expires)
}

// Copy duplicates a file within its bucket.
func (s *S3) Copy(ctx context.Context, destination, source string, userID int) error {
	return s.getGoAMZ().PutReaderHeader(destination, http.NoBody, 0, map[string][]string{
		"x-amz-copy-source": []string{url.PathEscape(path.Join("/"+s.BucketName, source))},
		//"Transfer-Encoding": []string{""},
		"Content-Length": []string{"0"},
	}, s3lib.Private)
}

// ListActions implements Source.
func (s *S3) ListActions(ctx context.Context, p string, userID int) ([]Action, error) {
	return []Action{
		{
			Kind:       ActionKindButton,
			ID:         "gen_1hr_url",
			Name:       "Generate fetch URL (1 hr)",
			OutputType: OutputTypeURL,
		},
		{
			Kind:       ActionKindUnaryStringInput,
			ID:         "copy",
			Name:       "Make a copy (relative path)",
			OutputType: OutputTypeNone,
		},
	}, nil
}

// RunAction implements Source.
func (s *S3) RunAction(ctx context.Context, p string, userID int, action string, payload map[string]string) ([]interface{}, error) {
	switch action {
	case "gen_1hr_url":
		return []interface{}{
			s.SignedURL(ctx, p, time.Now().Add(time.Hour), userID),
		}, nil
	case "copy":
		if _, ok := payload["1_string"]; !ok {
			return nil, errors.New("expected parameter: 1_string")
		}
		return nil, s.Copy(ctx, path.Join(path.Base(p), payload["1_string"]), p, userID)
	}
	return nil, errors.New("no such action")
}
