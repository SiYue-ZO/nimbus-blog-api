package persistence

import (
	"context"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type siteSettingRepo struct {
	query *query.Query
}

func NewSiteSettingRepo(db *gorm.DB) repo.SiteSettingRepo {
	return &siteSettingRepo{query: query.Use(db)}
}

func (r *siteSettingRepo) ListAll(ctx context.Context) ([]*entity.SiteSetting, error) {
	s := r.query.SiteSetting
	rows, err := s.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	settings := make([]*entity.SiteSetting, len(rows))
	for i, ms := range rows {
		settings[i] = toEntitySiteSetting(ms)
	}
	return settings, nil
}

func (r *siteSettingRepo) GetByKey(ctx context.Context, key string) (*entity.SiteSetting, error) {
	s := r.query.SiteSetting
	ms, err := s.WithContext(ctx).Where(s.SettingKey.Eq(key)).First()
	if err != nil {
		return nil, err
	}
	return toEntitySiteSetting(ms), nil
}

func (r *siteSettingRepo) Upsert(ctx context.Context, es entity.SiteSetting) error {
	ms := &model.SiteSetting{
		SettingKey:  es.SettingKey,
		SettingType: es.SettingType,
		Description: es.Description,
		IsPublic:    es.IsPublic,
	}
	if es.SettingValue != nil {
		ms.SettingValue = *es.SettingValue
	}
	s := r.query.SiteSetting
	return s.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"setting_value", "setting_type", "description", "is_public", "updated_at"}),
	}).Create(ms)
}

func toEntitySiteSetting(ms *model.SiteSetting) *entity.SiteSetting {
	es := &entity.SiteSetting{
		ID:          ms.ID,
		SettingKey:  ms.SettingKey,
		SettingType: ms.SettingType,
		Description: ms.Description,
		IsPublic:    ms.IsPublic,
		CreatedAt:   ms.CreatedAt,
		UpdatedAt:   ms.UpdatedAt,
	}
	if ms.SettingValue != "" {
		es.SettingValue = &ms.SettingValue
	}
	return es
}
