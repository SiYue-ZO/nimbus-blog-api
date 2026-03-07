package storage

import (
	"context"
	"fmt"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/scc749/nimbus-blog-api/internal/repo"
)

// minioStore implements repo.ObjectStore using MinIO Go SDK v7.
type minioStore struct {
	cli *minio.Client
}

func NewMinioStore(cli *minio.Client) repo.ObjectStore {
	return &minioStore{cli: cli}
}

func (s *minioStore) PresignUpload(ctx context.Context, bucket, key string, expires time.Duration, _ string) (string, error) {
	u, err := s.cli.PresignedPutObject(ctx, bucket, key, expires)
	if err != nil {
		return "", fmt.Errorf("MinioStore - PresignUpload - PresignedPutObject: %w", err)
	}
	return u.String(), nil
}

func (s *minioStore) PresignDownload(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	u, err := s.cli.PresignedGetObject(ctx, bucket, key, expires, nil)
	if err != nil {
		return "", fmt.Errorf("MinioStore - PresignDownload - PresignedGetObject: %w", err)
	}
	return u.String(), nil
}

func (s *minioStore) Delete(ctx context.Context, bucket, key string) error {
	if err := s.cli.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("MinioStore - Delete - RemoveObject: %w", err)
	}
	return nil
}
