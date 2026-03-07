package feedback

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
	feedbacks repo.FeedbackRepo
}

// New 创建 Feedback UseCase。
func New(feedbacks repo.FeedbackRepo) usecase.Feedback {
	return &useCase{feedbacks: feedbacks}
}

// Admin 管理端用例。

func (u *useCase) ListFeedbacks(ctx context.Context, params input.ListFeedbacks) (*output.ListResult[output.FeedbackDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var status *string
	if params.Status != nil {
		status = (*string)(params.Status)
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	feedbacks, total, err := u.feedbacks.List(ctx, offset, params.PageSize, status, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items := make([]output.FeedbackDetail, len(feedbacks))
	for i, f := range feedbacks {
		items[i] = toFeedbackDetail(f)
	}

	return &output.ListResult[output.FeedbackDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) GetFeedbackByID(ctx context.Context, id int64) (*output.FeedbackDetail, error) {
	f, err := u.feedbacks.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	detail := toFeedbackDetail(f)
	return &detail, nil
}

func (u *useCase) UpdateFeedback(ctx context.Context, params input.UpdateFeedback) error {
	if err := u.feedbacks.UpdateStatus(ctx, params.ID, params.Status); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func (u *useCase) DeleteFeedback(ctx context.Context, id int64) error {
	if err := u.feedbacks.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Public 公共端用例。

func (u *useCase) SubmitFeedback(ctx context.Context, params input.SubmitFeedback) error {
	f := entity.Feedback{
		Name:      params.Name,
		Email:     params.Email,
		Type:      params.Type,
		Subject:   params.Subject,
		Message:   params.Message,
		Status:    entity.FeedbackStatusPending,
		IPAddress: params.IPAddress,
		UserAgent: params.UserAgent,
	}
	if _, err := u.feedbacks.Create(ctx, f); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Helpers 辅助函数。

func toFeedbackDetail(f *entity.Feedback) output.FeedbackDetail {
	return output.FeedbackDetail{
		ID:        f.ID,
		Name:      f.Name,
		Email:     f.Email,
		Type:      f.Type,
		Subject:   f.Subject,
		Message:   f.Message,
		Status:    f.Status,
		IPAddress: f.IPAddress,
		UserAgent: f.UserAgent,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}
}
