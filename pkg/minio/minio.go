// Package minio MinIO Client 封装。
package minio

import (
	"context"
	"fmt"
	"log"
	"time"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	_defaultConnAttempts = 10
	_defaultConnTimeout  = time.Second
)

// Client MinIO Client 容器。
type Client struct {
	connAttempts  int
	connTimeout   time.Duration
	DefaultBucket string
	Region        string
	CLI           *minio.Client
}

// New 创建 MinIO Client。
func New(endpoint, accessKey, secretKey string, useSSL bool, opts ...Option) (*Client, error) {
	c := &Client{connAttempts: _defaultConnAttempts, connTimeout: _defaultConnTimeout}
	for _, opt := range opts {
		opt(c)
	}

	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
		Region: c.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio - New - init: %w", err)
	}
	c.CLI = cli

	attempts := c.connAttempts
	var lastErr error
	for attempts > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), c.connTimeout)
		if c.DefaultBucket != "" {
			_, lastErr = cli.BucketExists(ctx, c.DefaultBucket)
		} else {
			_, lastErr = cli.ListBuckets(ctx)
		}
		cancel()
		if lastErr == nil {
			break
		}
		log.Printf("MinIO: connect retry, attempts left: %d", attempts)
		time.Sleep(c.connTimeout)
		attempts--
	}
	if lastErr != nil {
		log.Printf("MinIO: connection check failed: %v", lastErr)
		return nil, fmt.Errorf("minio - New - connection check failed: %w", lastErr)
	}
	return c, nil
}

// EnsureBucket 确保 bucket 存在。
func (c *Client) EnsureBucket(ctx context.Context, bucket string) error {
	exists, err := c.CLI.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	opts := minio.MakeBucketOptions{Region: c.Region}
	return c.CLI.MakeBucket(ctx, bucket, opts)
}
