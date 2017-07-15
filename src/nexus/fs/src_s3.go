package fs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/mitchellh/goamz/aws"
	s3lib "github.com/mitchellh/goamz/s3"
	"github.com/rlmcpherson/s3gof3r"
)

type s3 struct {
	SecretKey  string
	AccessKey  string
	BucketName string
	RegionName string
}

func (s *s3) getGoAMZ() *s3lib.Bucket {
	s3cli := s3lib.New(aws.Auth{AccessKey: s.AccessKey, SecretKey: s.SecretKey, Token: ""}, aws.Regions[s.RegionName])
	return s3cli.Bucket(s.BucketName)
}

func (s *s3) getStreamingCredentials() *s3gof3r.S3 {
	keys := s3gof3r.Keys{AccessKey: s.AccessKey, SecretKey: s.SecretKey}
	return s3gof3r.New(strings.Replace(aws.Regions[s.RegionName].S3Endpoint, "https://", "", -1), keys)
}

func (s *s3) getStreamParameters() *s3gof3r.Config {
	return &s3gof3r.Config{
		Concurrency: 2,
		PartSize:    6 * 1024 * 1024,
		NTry:        10,
		Md5Check:    true,
		Scheme:      "https",
		Client:      s3gof3r.ClientWithTimeout(12 * time.Second),
	}
}

func (s *s3) Contents(ctx context.Context, p string, userID int, writer io.Writer) error {
	s3Reader, _, err := s.getStreamingCredentials().Bucket(s.BucketName).GetReader(p, s.getStreamParameters())
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, s3Reader)
	return err
}

func (s *s3) Save(ctx context.Context, p string, userID int, data []byte) error {
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

func (s *s3) Delete(ctx context.Context, p string, userID int) error {
	return s.getGoAMZ().Del(p)
}

func (s *s3) List(ctx context.Context, p string, userID int) ([]ListResultItem, error) {
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
			Name:     line.Key,
			ItemKind: KindFile,
			Modified: t,
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

func (s *s3) NewFolder(ctx context.Context, p string, userID int) error {
	return errors.New("not implemented")
}
