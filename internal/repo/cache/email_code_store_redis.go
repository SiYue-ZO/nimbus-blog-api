package cache

import (
	"context"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/pkg/redis"
)

// emailCodeRedisStore stores email verification codes in Redis with TTL.
type emailCodeRedisStore struct {
	rdb    *redis.Redis
	ttl    time.Duration
	prefix string
}

// NewEmailCodeRedisStore constructs a Redis-backed store with a fixed TTL.
func NewEmailCodeRedisStore(r *redis.Redis, ttl time.Duration) repo.EmailCodeStore {
	return &emailCodeRedisStore{rdb: r, ttl: ttl, prefix: "email_code:"}
}

// Set stores the verification code for the given id (email) with TTL.
func (s *emailCodeRedisStore) Set(id string, value string) error {
	ctx := context.Background()
	return s.rdb.RDB.Set(ctx, s.prefix+id, value, s.ttl).Err()
}

// Get retrieves the code; when clear is true, the key is deleted.
func (s *emailCodeRedisStore) Get(id string, clear bool) string {
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

// Verify checks the provided value against stored code.
func (s *emailCodeRedisStore) Verify(id, value string, clear bool) bool {
	v := s.Get(id, clear)
	return v == value
}
