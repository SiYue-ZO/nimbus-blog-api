package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RefreshTokenBlacklistPostgres struct {
	q *query.Query
}

func NewRefreshTokenBlacklistPostgres(db *gorm.DB) repo.RefreshTokenBlacklistRepo {
	return &RefreshTokenBlacklistPostgres{q: query.Use(db)}
}

func (r *RefreshTokenBlacklistPostgres) Add(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	item := &model.RefreshTokenBlacklist{
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}
	return r.q.RefreshTokenBlacklist.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(item)
}

func (r *RefreshTokenBlacklistPostgres) Exists(ctx context.Context, tokenHash string) (bool, error) {
	_, err := r.q.RefreshTokenBlacklist.WithContext(ctx).Where(r.q.RefreshTokenBlacklist.TokenHash.Eq(tokenHash)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
