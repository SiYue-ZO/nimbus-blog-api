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

type fileRepo struct {
	query *query.Query
}

func NewFileRepo(db *gorm.DB) repo.FileRepo {
	return &fileRepo{query: query.Use(db)}
}

func (r *fileRepo) Create(ctx context.Context, ef entity.File) (int64, error) {
	mf := toModelFile(&ef)
	if err := r.query.File.WithContext(ctx).Create(mf); err != nil {
		return 0, err
	}
	return mf.ID, nil
}

func (r *fileRepo) GetByObjectKey(ctx context.Context, objectKey string) (*entity.File, error) {
	f := r.query.File
	mf, err := f.WithContext(ctx).Where(f.ObjectKey.Eq(objectKey)).First()
	if err != nil {
		return nil, err
	}
	return toEntityFile(mf), nil
}

func (r *fileRepo) List(ctx context.Context, offset, limit int, usage *string, sortBy *string, order *string) ([]*entity.File, int64, error) {
	f := r.query.File
	q := f.WithContext(ctx)
	if usage != nil && *usage != "" {
		q = q.Where(f.Usage.Eq(*usage))
	}

	total, err := q.Count()
	if err != nil {
		return nil, 0, err
	}

	if sortBy != nil && *sortBy != "" {
		orderField, ok := f.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				q = q.Order(orderField)
			} else {
				q = q.Order(orderField.Desc())
			}
		}
	} else {
		q = q.Order(f.CreatedAt.Desc())
	}

	rows, err := q.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}
	files := make([]*entity.File, len(rows))
	for i, mf := range rows {
		files[i] = toEntityFile(mf)
	}
	return files, total, nil
}

func (r *fileRepo) ListByResource(ctx context.Context, usage string, resourceID int64) ([]*entity.File, error) {
	f := r.query.File
	rows, err := f.WithContext(ctx).Where(f.Usage.Eq(usage), f.ResourceID.Eq(resourceID)).Order(f.CreatedAt.Desc()).Find()
	if err != nil {
		return nil, err
	}
	files := make([]*entity.File, len(rows))
	for i, mf := range rows {
		files[i] = toEntityFile(mf)
	}
	return files, nil
}

func (r *fileRepo) UpdateResourceID(ctx context.Context, objectKey string, resourceID int64) error {
	f := r.query.File
	_, err := f.WithContext(ctx).Where(f.ObjectKey.Eq(objectKey)).Update(f.ResourceID, resourceID)
	return err
}

func (r *fileRepo) ClearResourceIDByResourceAndUsage(ctx context.Context, resourceID int64, usage string) error {
	f := r.query.File
	_, err := f.WithContext(ctx).Where(f.ResourceID.Eq(resourceID), f.Usage.Eq(usage)).UpdateSimple(f.ResourceID.Null())
	return err
}

func (r *fileRepo) DeleteByObjectKey(ctx context.Context, objectKey string) error {
	f := r.query.File
	_, err := f.WithContext(ctx).Where(f.ObjectKey.Eq(objectKey)).Delete()
	return err
}

func toModelFile(ef *entity.File) *model.File {
	mf := &model.File{
		ID:         ef.ID,
		ObjectKey:  ef.ObjectKey,
		FileName:   ef.FileName,
		FileSize:   ef.FileSize,
		MimeType:   ef.MimeType,
		Usage:      ef.Usage,
		UploaderID: ef.UploaderID,
		CreatedAt:  ef.CreatedAt,
	}
	if ef.ResourceID != nil {
		mf.ResourceID = ef.ResourceID
	}
	return mf
}

func toEntityFile(mf *model.File) *entity.File {
	return &entity.File{
		ID:         mf.ID,
		ObjectKey:  mf.ObjectKey,
		FileName:   mf.FileName,
		FileSize:   mf.FileSize,
		MimeType:   mf.MimeType,
		Usage:      mf.Usage,
		ResourceID: mf.ResourceID,
		UploaderID: mf.UploaderID,
		CreatedAt:  mf.CreatedAt,
	}
}
