package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
	"gorm.io/gorm"
)

var (
	// ErrRepo Repo 错误哨兵。
	ErrRepo = errors.New("repo")
	// ErrNotFound NotFound 错误哨兵。
	ErrNotFound = errors.New("not found")
)

type useCase struct {
	users repo.UserRepo
}

// New 创建 User UseCase。
func New(users repo.UserRepo) usecase.User {
	return &useCase{users: users}
}

// Admin 管理端用例。

func (u *useCase) ListUsers(ctx context.Context, params input.ListUsers) (*output.ListResult[output.UserDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var status *string
	if params.Status != nil {
		status = (*string)(params.Status)
	}
	var keyword *string
	if params.Keyword != nil {
		keyword = &params.Keyword.Keyword
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	users, total, err := u.users.List(ctx, offset, params.PageSize, status, keyword, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items := make([]output.UserDetail, len(users))
	for i, eu := range users {
		items[i] = toUserDetail(eu)
	}

	return &output.ListResult[output.UserDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) UpdateStatus(ctx context.Context, id int64, status string) error {
	if err := u.users.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Public 公共端用例。

func (u *useCase) GetUserByID(ctx context.Context, id int64) (*output.UserDetail, error) {
	eu, err := u.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %w", ErrNotFound, err)
		}
		return nil, fmt.Errorf("%w: %w", ErrRepo, err)
	}
	detail := toUserDetail(eu)
	return &detail, nil
}

func (u *useCase) UpdateProfile(ctx context.Context, id int64, params input.UpdateProfile) error {
	eu := entity.User{
		ID:              id,
		Name:            params.Name,
		Bio:             params.Bio,
		Region:          &params.Region,
		BlogURL:         &params.BlogURL,
		ShowFullProfile: params.ShowFullProfile,
	}
	if err := u.users.Update(ctx, eu); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Helpers 辅助函数。

func toUserDetail(eu *entity.User) output.UserDetail {
	return output.UserDetail{
		ID:              eu.ID,
		Name:            eu.Name,
		Email:           eu.Email,
		Avatar:          eu.Avatar,
		Bio:             eu.Bio,
		Status:          eu.Status,
		EmailVerified:   eu.EmailVerified,
		Region:          eu.Region,
		BlogURL:         eu.BlogURL,
		AuthProvider:    eu.AuthProvider,
		AuthOpenid:      eu.AuthOpenid,
		ShowFullProfile: eu.ShowFullProfile,
		CreatedAt:       eu.CreatedAt,
		UpdatedAt:       eu.UpdatedAt,
	}
}
