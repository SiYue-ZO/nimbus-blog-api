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

type linkRepo struct {
	query *query.Query
}

func NewLinkRepo(db *gorm.DB) repo.LinkRepo {
	return &linkRepo{query: query.Use(db)}
}

func (r *linkRepo) List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string) ([]*entity.Link, int64, error) {
	l := r.query.Link
	do := l.WithContext(ctx)

	if keyword != nil && *keyword != "" {
		do = do.Where(field.NewUnsafeFieldRaw("name ILIKE ? OR description ILIKE ?", "%"+*keyword+"%", "%"+*keyword+"%"))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if keyword != nil && *keyword != "" {
		do = do.Order(field.NewUnsafeFieldRaw("similarity(name || ' ' || COALESCE(description, ''), ?)", *keyword).Desc())
	} else if sortBy != nil && *sortBy != "" {
		orderField, ok := l.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else {
		do = do.Order(l.SortOrder.Asc(), l.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	links := make([]*entity.Link, len(rows))
	for i, ml := range rows {
		links[i] = toEntityLink(ml)
	}
	return links, total, nil
}

func (r *linkRepo) Create(ctx context.Context, el entity.Link) (int64, error) {
	ml := toModelLink(&el)
	if err := r.query.Link.WithContext(ctx).Create(ml); err != nil {
		return 0, err
	}
	return ml.ID, nil
}

func (r *linkRepo) Update(ctx context.Context, el entity.Link) error {
	l := r.query.Link
	_, err := l.WithContext(ctx).Where(l.ID.Eq(el.ID)).Updates(toModelLink(&el))
	return err
}

func (r *linkRepo) Delete(ctx context.Context, id int64) error {
	l := r.query.Link
	_, err := l.WithContext(ctx).Where(l.ID.Eq(id)).Delete()
	return err
}

func (r *linkRepo) ListAllPublic(ctx context.Context) ([]*entity.Link, error) {
	l := r.query.Link
	rows, err := l.WithContext(ctx).Where(l.Status.Eq("active")).Order(l.SortOrder.Asc(), l.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, err
	}
	links := make([]*entity.Link, len(rows))
	for i, ml := range rows {
		links[i] = toEntityLink(ml)
	}
	return links, nil
}

func toModelLink(el *entity.Link) *model.Link {
	ml := &model.Link{
		ID:        el.ID,
		Name:      el.Name,
		URL:       el.URL,
		SortOrder: el.SortOrder,
		Status:    el.Status,
		CreatedAt: el.CreatedAt,
		UpdatedAt: el.UpdatedAt,
	}
	if el.Logo != nil {
		ml.Logo = *el.Logo
	}
	if el.Description != nil {
		ml.Description = *el.Description
	}
	return ml
}

func toEntityLink(ml *model.Link) *entity.Link {
	el := &entity.Link{
		ID:        ml.ID,
		Name:      ml.Name,
		URL:       ml.URL,
		SortOrder: ml.SortOrder,
		Status:    ml.Status,
		CreatedAt: ml.CreatedAt,
		UpdatedAt: ml.UpdatedAt,
	}
	if ml.Logo != "" {
		el.Logo = &ml.Logo
	}
	if ml.Description != "" {
		el.Description = &ml.Description
	}
	return el
}
