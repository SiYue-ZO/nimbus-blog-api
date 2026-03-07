package minio

import "time"

// Option MinIO Client 选项。
type Option func(*Client)

// WithConnAttempts 配置连接重试次数。
func WithConnAttempts(attempts int) Option {
	return func(c *Client) {
		c.connAttempts = attempts
	}
}

// WithConnTimeout 配置连接重试间隔。
func WithConnTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.connTimeout = timeout
	}
}

// WithDefaultBucket 配置默认 bucket。
func WithDefaultBucket(bucket string) Option {
	return func(c *Client) {
		c.DefaultBucket = bucket
	}
}

// WithRegion 配置 Region。
func WithRegion(region string) Option {
	return func(c *Client) {
		c.Region = region
	}
}
