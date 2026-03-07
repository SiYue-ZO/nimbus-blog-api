package cache

import (
	"context"
	"time"

	json "github.com/goccy/go-json"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/pkg/redis"
)

type adminTwoFASetupValue struct {
	AdminID int64  `json:"admin_id"`
	Secret  string `json:"secret"`
}

type adminTwoFARedisStore struct {
	rdb    *redis.Redis
	prefix string
}

func NewAdminTwoFARedisStore(r *redis.Redis) repo.AdminTwoFASetupStore {
	return &adminTwoFARedisStore{rdb: r, prefix: "admin_2fa_setup:"}
}

func (s *adminTwoFARedisStore) Set(setupID string, adminID int64, secret string, ttl time.Duration) error {
	ctx := context.Background()
	key := s.prefix + setupID
	bs, err := json.Marshal(adminTwoFASetupValue{AdminID: adminID, Secret: secret})
	if err != nil {
		return err
	}
	return s.rdb.RDB.Set(ctx, key, bs, ttl).Err()
}

func (s *adminTwoFARedisStore) Get(setupID string) (adminID int64, secret string, ok bool) {
	ctx := context.Background()
	key := s.prefix + setupID
	val, err := s.rdb.RDB.Get(ctx, key).Bytes()
	if err != nil {
		return 0, "", false
	}
	var v adminTwoFASetupValue
	if err := json.Unmarshal(val, &v); err != nil {
		return 0, "", false
	}
	if v.AdminID <= 0 || v.Secret == "" {
		return 0, "", false
	}
	return v.AdminID, v.Secret, true
}

func (s *adminTwoFARedisStore) Delete(setupID string) error {
	ctx := context.Background()
	return s.rdb.RDB.Del(ctx, s.prefix+setupID).Err()
}
