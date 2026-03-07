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

type commentRepo struct {
	query *query.Query
}

func NewCommentRepo(db *gorm.DB) repo.CommentRepo {
	return &commentRepo{query: query.Use(db)}
}

func (r *commentRepo) List(ctx context.Context, offset, limit int, status *string, sortBy *string, order *string) ([]*entity.Comment, int64, error) {
	c := r.query.Comment
	do := c.WithContext(ctx)

	if status != nil && *status != "" {
		do = do.Where(c.Status.Eq(*status))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if sortBy != nil && *sortBy != "" {
		orderField, ok := c.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else {
		do = do.Order(c.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	comments := make([]*entity.Comment, len(rows))
	for i, mc := range rows {
		comments[i] = toEntityComment(mc)
	}
	return comments, total, nil
}

func (r *commentRepo) GetByID(ctx context.Context, id int64) (*entity.Comment, error) {
	c := r.query.Comment
	mc, err := c.WithContext(ctx).Where(c.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return toEntityComment(mc), nil
}

func (r *commentRepo) Create(ctx context.Context, ec entity.Comment) (int64, error) {
	mc := toModelComment(&ec)
	if err := r.query.Comment.WithContext(ctx).Create(mc); err != nil {
		return 0, err
	}
	return mc.ID, nil
}

func (r *commentRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	c := r.query.Comment
	_, err := c.WithContext(ctx).Where(c.ID.Eq(id)).Update(c.Status, status)
	return err
}

func (r *commentRepo) Delete(ctx context.Context, id int64) error {
	c := r.query.Comment
	_, err := c.WithContext(ctx).Where(c.ID.Eq(id)).Delete()
	return err
}

func (r *commentRepo) ListApprovedByPostID(ctx context.Context, postID int64) ([]*entity.Comment, error) {
	c := r.query.Comment
	rows, err := c.WithContext(ctx).
		Where(c.PostID.Eq(postID), c.Status.Eq("approved")).
		Order(c.CreatedAt.Asc()).
		Find()
	if err != nil {
		return nil, err
	}
	comments := make([]*entity.Comment, len(rows))
	for i, mc := range rows {
		comments[i] = toEntityComment(mc)
	}
	return comments, nil
}

func toModelComment(ec *entity.Comment) *model.Comment {
	return &model.Comment{
		ID:        ec.ID,
		PostID:    ec.PostID,
		ParentID:  ec.ParentID,
		UserID:    ec.UserID,
		Content:   ec.Content,
		Status:    ec.Status,
		Likes:     ec.Likes,
		IPAddress: ec.IPAddress,
		UserAgent: ec.UserAgent,
		CreatedAt: ec.CreatedAt,
		UpdatedAt: ec.UpdatedAt,
	}
}

func toEntityComment(mc *model.Comment) *entity.Comment {
	return &entity.Comment{
		ID:        mc.ID,
		PostID:    mc.PostID,
		ParentID:  mc.ParentID,
		UserID:    mc.UserID,
		Content:   mc.Content,
		Status:    mc.Status,
		Likes:     mc.Likes,
		IPAddress: mc.IPAddress,
		UserAgent: mc.UserAgent,
		CreatedAt: mc.CreatedAt,
		UpdatedAt: mc.UpdatedAt,
	}
}
