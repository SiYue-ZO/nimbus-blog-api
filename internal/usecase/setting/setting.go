package setting

import (
	"context"
	"errors"
	"fmt"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
)

var (
	// ErrRepo Repo 错误哨兵。
	ErrRepo = errors.New("repo")
	// ErrNotFound NotFound 错误哨兵。
	ErrNotFound = errors.New("not found")
)

type useCase struct {
	settings repo.SiteSettingRepo
}

// New 创建 Setting UseCase。
func New(settings repo.SiteSettingRepo) usecase.Setting {
	return &useCase{settings: settings}
}

// Admin 管理端用例。

func (u *useCase) GetAllSiteSettings(ctx context.Context) (*output.AllResult[output.SiteSettingDetail], error) {
	settings, err := u.settings.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	items := make([]output.SiteSettingDetail, len(settings))
	for i, s := range settings {
		items[i] = toSiteSettingDetail(s)
	}
	return &output.AllResult[output.SiteSettingDetail]{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

func (u *useCase) GetSiteSettingByKey(ctx context.Context, key string) (*output.SiteSettingDetail, error) {
	s, err := u.settings.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	detail := toSiteSettingDetail(s)
	return &detail, nil
}

func (u *useCase) UpsertSiteSetting(ctx context.Context, params input.UpsertSiteSetting) error {
	s := entity.SiteSetting{
		SettingKey:   params.SettingKey,
		SettingValue: params.SettingValue,
		SettingType:  params.SettingType,
		Description:  params.Description,
		IsPublic:     params.IsPublic,
	}
	if err := u.settings.Upsert(ctx, s); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Helpers 辅助函数。

func toSiteSettingDetail(s *entity.SiteSetting) output.SiteSettingDetail {
	return output.SiteSettingDetail{
		ID:           s.ID,
		SettingKey:   s.SettingKey,
		SettingValue: s.SettingValue,
		SettingType:  s.SettingType,
		Description:  s.Description,
		IsPublic:     s.IsPublic,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}
