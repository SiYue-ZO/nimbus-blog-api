package link

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
	links repo.LinkRepo
}

// New 创建 Link UseCase。
func New(links repo.LinkRepo) usecase.Link {
	return &useCase{links: links}
}

// Admin 管理端用例。

func (u *useCase) ListLinks(ctx context.Context, params input.ListLinks) (*output.ListResult[output.LinkDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var keyword *string
	if params.Keyword != nil {
		keyword = &params.Keyword.Keyword
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	links, total, err := u.links.List(ctx, offset, params.PageSize, keyword, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items := make([]output.LinkDetail, len(links))
	for i, l := range links {
		items[i] = toLinkDetail(l)
	}

	return &output.ListResult[output.LinkDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) CreateLink(ctx context.Context, params input.CreateLink) (int64, error) {
	l := entity.Link{
		Name:        params.Name,
		URL:         params.URL,
		Description: params.Description,
		Logo:        params.Logo,
		SortOrder:   params.SortOrder,
		Status:      params.Status,
	}
	id, err := u.links.Create(ctx, l)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return id, nil
}

func (u *useCase) UpdateLink(ctx context.Context, params input.UpdateLink) error {
	l := entity.Link{
		ID:          params.ID,
		Name:        params.Name,
		URL:         params.URL,
		Description: params.Description,
		Logo:        params.Logo,
		SortOrder:   params.SortOrder,
		Status:      params.Status,
	}
	if err := u.links.Update(ctx, l); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func (u *useCase) DeleteLink(ctx context.Context, id int64) error {
	if err := u.links.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Public 公共端用例。

func (u *useCase) GetAllPublicLinks(ctx context.Context) (*output.AllResult[output.LinkDetail], error) {
	links, err := u.links.ListAllPublic(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	items := make([]output.LinkDetail, len(links))
	for i, l := range links {
		items[i] = toLinkDetail(l)
	}
	return &output.AllResult[output.LinkDetail]{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// Helpers 辅助函数。

func toLinkDetail(l *entity.Link) output.LinkDetail {
	return output.LinkDetail{
		ID:          l.ID,
		Name:        l.Name,
		URL:         l.URL,
		Description: l.Description,
		Logo:        l.Logo,
		SortOrder:   l.SortOrder,
		Status:      l.Status,
		CreatedAt:   l.CreatedAt,
		UpdatedAt:   l.UpdatedAt,
	}
}
