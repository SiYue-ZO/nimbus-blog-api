package persistence

import (
	"context"
	"strings"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type categoryRepo struct {
	query *query.Query
}

func NewCategoryRepo(db *gorm.DB) repo.CategoryRepo {
	return &categoryRepo{query: query.Use(db)}
}

func (r *categoryRepo) List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string) ([]*entity.Category, int64, error) {
	c := r.query.Category
	do := c.WithContext(ctx)

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

	categories := make([]*entity.Category, len(rows))
	for i, mc := range rows {
		categories[i] = toEntityCategory(mc)
	}
	return categories, total, nil
}

func (r *categoryRepo) ListAll(ctx context.Context) ([]*entity.Category, error) {
	c := r.query.Category
	rows, err := c.WithContext(ctx).Order(c.PostCount.Desc()).Find()
	if err != nil {
		return nil, err
	}
	categories := make([]*entity.Category, len(rows))
	for i, mc := range rows {
		categories[i] = toEntityCategory(mc)
	}
	return categories, nil
}

func (r *categoryRepo) GetByID(ctx context.Context, id int64) (*entity.Category, error) {
	c := r.query.Category
	mc, err := c.WithContext(ctx).Where(c.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return toEntityCategory(mc), nil
}

func (r *categoryRepo) GetBySlug(ctx context.Context, slug string) (*entity.Category, error) {
	c := r.query.Category
	mc, err := c.WithContext(ctx).Where(c.Slug.Eq(slug)).First()
	if err != nil {
		return nil, err
	}
	return toEntityCategory(mc), nil
}

func (r *categoryRepo) Create(ctx context.Context, ec entity.Category) (int64, error) {
	mc := &model.Category{Name: ec.Name, Slug: ec.Slug}
	if err := r.query.Category.WithContext(ctx).Create(mc); err != nil {
		return 0, err
	}
	return mc.ID, nil
}

func (r *categoryRepo) Update(ctx context.Context, ec entity.Category) error {
	c := r.query.Category
	_, err := c.WithContext(ctx).Where(c.ID.Eq(ec.ID)).Updates(&model.Category{Name: ec.Name, Slug: ec.Slug})
	return err
}

func (r *categoryRepo) Delete(ctx context.Context, id int64) error {
	c := r.query.Category
	_, err := c.WithContext(ctx).Where(c.ID.Eq(id)).Delete()
	return err
}

func toEntityCategory(mc *model.Category) *entity.Category {
	return &entity.Category{
		ID:        mc.ID,
		Name:      mc.Name,
		Slug:      mc.Slug,
		PostCount: mc.PostCount,
		CreatedAt: mc.CreatedAt,
		UpdatedAt: mc.UpdatedAt,
	}
}
