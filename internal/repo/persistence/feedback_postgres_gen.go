package persistence

import (
	"context"
	"strings"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gorm"
)

type feedbackRepo struct {
	query *query.Query
}

func NewFeedbackRepo(db *gorm.DB) repo.FeedbackRepo {
	return &feedbackRepo{query: query.Use(db)}
}

func (r *feedbackRepo) List(ctx context.Context, offset, limit int, status *string, sortBy *string, order *string) ([]*entity.Feedback, int64, error) {
	f := r.query.Feedback
	do := f.WithContext(ctx)

	if status != nil && *status != "" {
		do = do.Where(f.Status.Eq(*status))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if sortBy != nil && *sortBy != "" {
		orderField, ok := f.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else {
		do = do.Order(f.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	feedbacks := make([]*entity.Feedback, len(rows))
	for i, mf := range rows {
		feedbacks[i] = toEntityFeedback(mf)
	}
	return feedbacks, total, nil
}

func (r *feedbackRepo) GetByID(ctx context.Context, id int64) (*entity.Feedback, error) {
	f := r.query.Feedback
	mf, err := f.WithContext(ctx).Where(f.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return toEntityFeedback(mf), nil
}

func (r *feedbackRepo) Create(ctx context.Context, ef entity.Feedback) (int64, error) {
	mf := toModelFeedback(&ef)
	if err := r.query.Feedback.WithContext(ctx).Create(mf); err != nil {
		return 0, err
	}
	return mf.ID, nil
}

func (r *feedbackRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	f := r.query.Feedback
	_, err := f.WithContext(ctx).Where(f.ID.Eq(id)).Update(f.Status, status)
	return err
}

func (r *feedbackRepo) Delete(ctx context.Context, id int64) error {
	f := r.query.Feedback
	_, err := f.WithContext(ctx).Where(f.ID.Eq(id)).Delete()
	return err
}

func toModelFeedback(ef *entity.Feedback) *model.Feedback {
	return &model.Feedback{
		ID:        ef.ID,
		Name:      ef.Name,
		Email:     ef.Email,
		Type:      ef.Type,
		Subject:   ef.Subject,
		Message:   ef.Message,
		Status:    ef.Status,
		IPAddress: ef.IPAddress,
		UserAgent: ef.UserAgent,
		CreatedAt: ef.CreatedAt,
		UpdatedAt: ef.UpdatedAt,
	}
}

func toEntityFeedback(mf *model.Feedback) *entity.Feedback {
	return &entity.Feedback{
		ID:        mf.ID,
		Name:      mf.Name,
		Email:     mf.Email,
		Type:      mf.Type,
		Subject:   mf.Subject,
		Message:   mf.Message,
		Status:    mf.Status,
		IPAddress: mf.IPAddress,
		UserAgent: mf.UserAgent,
		CreatedAt: mf.CreatedAt,
		UpdatedAt: mf.UpdatedAt,
	}
}
