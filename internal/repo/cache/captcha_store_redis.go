package cache

import (
	"context"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/pkg/redis"
)

// captchaRedisStore implements base64Captcha.Store backed by Redis with TTL
// It stores captcha values under key prefix "captcha:" and supports clear-on-read.
type captchaRedisStore struct {
	rdb    *redis.Redis
	ttl    time.Duration
	prefix string
}

// NewCaptchaRedisStore constructs a Redis-backed store with a fixed TTL.
func NewCaptchaRedisStore(r *redis.Redis, ttl time.Duration) repo.CaptchaStore {
	return &captchaRedisStore{rdb: r, ttl: ttl, prefix: "captcha:"}
}

// Set stores the captcha value for the given id with TTL.
func (s *captchaRedisStore) Set(id string, value string) error {
	ctx := context.Background()
	return s.rdb.RDB.Set(ctx, s.prefix+id, value, s.ttl).Err()
}

// Get retrieves the captcha value; when clear is true, the key is deleted.
func (s *captchaRedisStore) Get(id string, clear bool) string {
	ctx := context.Background()
	key := s.prefix + id
	val, err := s.rdb.RDB.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	if clear {
		_ = s.rdb.RDB.Del(ctx, key).Err()
	}
	return val
}

// Verify checks the answer against stored value, optionally clearing the key.
func (s *captchaRedisStore) Verify(id, answer string, clear bool) bool {
	v := s.Get(id, clear)
	return v == answer
}
