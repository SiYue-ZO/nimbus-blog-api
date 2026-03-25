package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/scc749/nimbus-blog-api/internal/repo"
)

// minioStore implements repo.ObjectStore using MinIO Go SDK v7.
type minioStore struct {
	cli           *minio.Client
	publicBaseURL string
}

func NewMinioStore(cli *minio.Client, publicBaseURL string) repo.ObjectStore {
	return &minioStore{cli: cli, publicBaseURL: publicBaseURL}
}

func (s *minioStore) PresignUpload(ctx context.Context, bucket, key string, expires time.Duration, _ string) (string, error) {
	u, err := s.cli.PresignedPutObject(ctx, bucket, key, expires)
	if err != nil {
		return "", fmt.Errorf("MinioStore - PresignUpload - PresignedPutObject: %w", err)
	}
	return rewriteToPublicBaseURL(u, s.publicBaseURL)
}

func (s *minioStore) PresignDownload(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	u, err := s.cli.PresignedGetObject(ctx, bucket, key, expires, nil)
	if err != nil {
		return "", fmt.Errorf("MinioStore - PresignDownload - PresignedGetObject: %w", err)
	}
	return rewriteToPublicBaseURL(u, s.publicBaseURL)
}

func (s *minioStore) Delete(ctx context.Context, bucket, key string) error {
	if err := s.cli.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("MinioStore - Delete - RemoveObject: %w", err)
	}
	return nil
}

func rewriteToPublicBaseURL(raw *url.URL, publicBaseURL string) (string, error) {
	if publicBaseURL == "" {
		return raw.String(), nil
	}
	base, err := url.Parse(publicBaseURL)
	if err != nil {
		return "", fmt.Errorf("MinioStore - rewriteToPublicBaseURL - parse public_base_url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("MinioStore - rewriteToPublicBaseURL - invalid public_base_url")
	}

	u := *base
	u.Path = joinURLPath(base.Path, raw.Path)
	u.RawQuery = raw.RawQuery
	if base.RawQuery != "" {
		if u.RawQuery != "" {
			u.RawQuery = base.RawQuery + "&" + u.RawQuery
		} else {
			u.RawQuery = base.RawQuery
		}
	}
	return u.String(), nil
}

func joinURLPath(prefix, p string) string {
	if prefix == "" {
		if p == "" {
			return "/"
		}
		if strings.HasPrefix(p, "/") {
			return p
		}
		return "/" + p
	}
	if p == "" {
		return prefix
	}
	return strings.TrimRight(prefix, "/") + "/" + strings.TrimLeft(p, "/")
}
