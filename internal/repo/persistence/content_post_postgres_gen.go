package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type postRepo struct {
	query *query.Query
}

func NewPostRepo(db *gorm.DB) repo.PostRepo {
	return &postRepo{query: query.Use(db)}
}

func (r *postRepo) List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string, categoryID *int, tagID *int, status *string, isFeatured *bool, featuredFirst bool) ([]*entity.Post, int64, error) {
	p := r.query.Post
	do := p.WithContext(ctx)

	if keyword != nil && *keyword != "" {
		kw := "%" + *keyword + "%"
		do = do.Where(field.NewUnsafeFieldRaw("title ILIKE ? OR COALESCE(excerpt, '') ILIKE ? OR content ILIKE ?", kw, kw, kw))
	}
	if categoryID != nil {
		do = do.Where(p.CategoryID.Eq(int64(*categoryID)))
	}
	if tagID != nil {
		pt := r.query.PostTag
		subQ := pt.WithContext(ctx).Where(pt.TagID.Eq(int64(*tagID)), pt.PostID.EqCol(p.ID))
		do = do.Where(gen.Exists(subQ))
	}
	if status != nil && *status != "" {
		do = do.Where(p.Status.Eq(*status))
	}
	if isFeatured != nil {
		do = do.Where(p.IsFeatured.Is(*isFeatured))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if keyword != nil && *keyword != "" {
		do = do.Order(field.NewUnsafeFieldRaw("similarity(title || ' ' || COALESCE(excerpt, '') || ' ' || content, ?)", *keyword).Desc())
	} else if sortBy != nil && *sortBy != "" {
		orderField, ok := p.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else if featuredFirst {
		do = do.Order(p.IsFeatured.Desc(), p.PublishedAt.Desc())
	} else {
		do = do.Order(p.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	posts := make([]*entity.Post, len(rows))
	for i, r := range rows {
		posts[i] = toEntityPost(r)
	}
	return posts, total, nil
}

func (r *postRepo) GetByID(ctx context.Context, id int64) (*entity.Post, error) {
	p := r.query.Post
	mp, err := p.WithContext(ctx).Where(p.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return toEntityPost(mp), nil
}

func (r *postRepo) GetBySlug(ctx context.Context, slug string) (*entity.Post, error) {
	p := r.query.Post
	mp, err := p.WithContext(ctx).Where(p.Slug.Eq(slug)).First()
	if err != nil {
		return nil, err
	}
	return toEntityPost(mp), nil
}

func (r *postRepo) Create(ctx context.Context, ep entity.Post) (int64, error) {
	mp := toModelPost(&ep)
	if err := r.query.Post.WithContext(ctx).Create(mp); err != nil {
		return 0, err
	}
	return mp.ID, nil
}

func (r *postRepo) Update(ctx context.Context, ep entity.Post) error {
	p := r.query.Post
	mp := toModelPost(&ep)
	tx := p.WithContext(ctx).Where(p.ID.Eq(ep.ID))
	_, err := tx.Updates(mp)
	if err != nil {
		return err
	}
	if ep.FeaturedImage == nil {
		_, err = tx.UpdateSimple(p.FeaturedImage.Null())
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *postRepo) Delete(ctx context.Context, id int64) error {
	p := r.query.Post
	_, err := p.WithContext(ctx).Where(p.ID.Eq(id)).Delete()
	return err
}

func (r *postRepo) SetTags(ctx context.Context, postID int64, tagIDs []int64) error {
	return r.query.Transaction(func(tx *query.Query) error {
		pt := tx.PostTag
		if _, err := pt.WithContext(ctx).Where(pt.PostID.Eq(postID)).Delete(); err != nil {
			return err
		}
		if len(tagIDs) == 0 {
			return nil
		}
		tags := make([]*model.PostTag, len(tagIDs))
		for i, tid := range tagIDs {
			tags[i] = &model.PostTag{PostID: postID, TagID: tid}
		}
		return pt.WithContext(ctx).Create(tags...)
	})
}

func toModelPost(p *entity.Post) *model.Post {
	mp := &model.Post{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		Content:     p.Content,
		AuthorID:    p.AuthorID,
		CategoryID:  p.CategoryID,
		Status:      p.Status,
		Views:       p.Views,
		Likes:       p.Likes,
		IsFeatured:  p.IsFeatured,
		PublishedAt: p.PublishedAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.Excerpt != nil {
		mp.Excerpt = *p.Excerpt
	}
	if p.FeaturedImage != nil {
		mp.FeaturedImage = p.FeaturedImage
	}
	if p.ReadTime != nil {
		mp.ReadTime = *p.ReadTime
	}
	if p.MetaTitle != nil {
		mp.MetaTitle = p.MetaTitle
	}
	if p.MetaDescription != nil {
		mp.MetaDescription = p.MetaDescription
	}
	return mp
}

func toEntityPost(mp *model.Post) *entity.Post {
	p := &entity.Post{
		ID:              mp.ID,
		Title:           mp.Title,
		Slug:            mp.Slug,
		Content:         mp.Content,
		FeaturedImage:   mp.FeaturedImage,
		AuthorID:        mp.AuthorID,
		CategoryID:      mp.CategoryID,
		Status:          mp.Status,
		Views:           mp.Views,
		Likes:           mp.Likes,
		IsFeatured:      mp.IsFeatured,
		MetaTitle:       mp.MetaTitle,
		MetaDescription: mp.MetaDescription,
		PublishedAt:     mp.PublishedAt,
		CreatedAt:       mp.CreatedAt,
		UpdatedAt:       mp.UpdatedAt,
	}
	if mp.Excerpt != "" {
		p.Excerpt = &mp.Excerpt
	}
	if mp.ReadTime != "" {
		p.ReadTime = &mp.ReadTime
	}
	return p
}

// suppress unused import
var _ = errors.Is
