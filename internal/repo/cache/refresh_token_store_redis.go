package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/pkg/redis"
)

type refreshTokenRedisStore struct {
	rdb    *redis.Redis
	prefix string
}

func NewRefreshTokenRedisStore(r *redis.Redis) repo.RefreshTokenStore {
	return &refreshTokenRedisStore{rdb: r, prefix: "refresh_token:"}
}

func (s *refreshTokenRedisStore) Set(userID int64, token string, ttl time.Duration) error {
	ctx := context.Background()
	key := s.prefix + strconv.FormatInt(userID, 10)
	return s.rdb.RDB.Set(ctx, key, token, ttl).Err()
}

func (s *refreshTokenRedisStore) Get(userID int64) string {
	ctx := context.Background()
	key := s.prefix + strconv.FormatInt(userID, 10)
	val, err := s.rdb.RDB.Get(ctx, key).Result()
	if err != nil {
		return ""
	}
	return val
}

func (s *refreshTokenRedisStore) Delete(userID int64) error {
	ctx := context.Background()
	key := s.prefix + strconv.FormatInt(userID, 10)
	return s.rdb.RDB.Del(ctx, key).Err()
}
