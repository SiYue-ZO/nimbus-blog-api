package persistence

import (
	"context"
	"strings"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type tagRepo struct {
	query *query.Query
}

func NewTagRepo(db *gorm.DB) repo.TagRepo {
	return &tagRepo{query: query.Use(db)}
}

func (r *tagRepo) List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string) ([]*entity.Tag, int64, error) {
	t := r.query.Tag
	do := t.WithContext(ctx)

	if keyword != nil && *keyword != "" {
		do = do.Where(field.NewUnsafeFieldRaw("name ILIKE ?", "%"+*keyword+"%"))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if keyword != nil && *keyword != "" {
		do = do.Order(field.NewUnsafeFieldRaw("similarity(name, ?)", *keyword).Desc())
	} else if sortBy != nil && *sortBy != "" {
		orderField, ok := t.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else {
		do = do.Order(t.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	tags := make([]*entity.Tag, len(rows))
	for i, mt := range rows {
		tags[i] = toEntityTag(mt)
	}
	return tags, total, nil
}

func (r *tagRepo) ListAll(ctx context.Context) ([]*entity.Tag, error) {
	t := r.query.Tag
	rows, err := t.WithContext(ctx).Order(t.PostCount.Desc()).Find()
	if err != nil {
		return nil, err
	}
	tags := make([]*entity.Tag, len(rows))
	for i, mt := range rows {
		tags[i] = toEntityTag(mt)
	}
	return tags, nil
}

func (r *tagRepo) GetByID(ctx context.Context, id int64) (*entity.Tag, error) {
	t := r.query.Tag
	mt, err := t.WithContext(ctx).Where(t.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return toEntityTag(mt), nil
}

func (r *tagRepo) GetBySlug(ctx context.Context, slug string) (*entity.Tag, error) {
	t := r.query.Tag
	mt, err := t.WithContext(ctx).Where(t.Slug.Eq(slug)).First()
	if err != nil {
		return nil, err
	}
	return toEntityTag(mt), nil
}

func (r *tagRepo) ListByPostID(ctx context.Context, postID int64) ([]*entity.Tag, error) {
	t := r.query.Tag
	pt := r.query.PostTag

	subQ := pt.WithContext(ctx).Where(pt.PostID.Eq(postID), pt.TagID.EqCol(t.ID))
	rows, err := t.WithContext(ctx).Where(gen.Exists(subQ)).Find()
	if err != nil {
		return nil, err
	}
	tags := make([]*entity.Tag, len(rows))
	for i, mt := range rows {
		tags[i] = toEntityTag(mt)
	}
	return tags, nil
}

func (r *tagRepo) Create(ctx context.Context, et entity.Tag) (int64, error) {
	mt := &model.Tag{Name: et.Name, Slug: et.Slug}
	if err := r.query.Tag.WithContext(ctx).Create(mt); err != nil {
		return 0, err
	}
	return mt.ID, nil
}

func (r *tagRepo) Update(ctx context.Context, et entity.Tag) error {
	t := r.query.Tag
	_, err := t.WithContext(ctx).Where(t.ID.Eq(et.ID)).Updates(&model.Tag{Name: et.Name, Slug: et.Slug})
	return err
}

func (r *tagRepo) Delete(ctx context.Context, id int64) error {
	t := r.query.Tag
	_, err := t.WithContext(ctx).Where(t.ID.Eq(id)).Delete()
	return err
}

func toEntityTag(mt *model.Tag) *entity.Tag {
	return &entity.Tag{
		ID:        mt.ID,
		Name:      mt.Name,
		Slug:      mt.Slug,
		PostCount: mt.PostCount,
		CreatedAt: mt.CreatedAt,
		UpdatedAt: mt.UpdatedAt,
	}
}
