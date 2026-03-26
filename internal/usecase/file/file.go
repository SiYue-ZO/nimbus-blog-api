package file

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
)

var (
	// ErrStorage Storage 错误哨兵。
	ErrStorage = errors.New("storage")
	// ErrDatabase Database 错误哨兵。
	ErrDatabase = errors.New("database")
)

type useCase struct {
	objects  repo.ObjectStore
	fileRepo repo.FileRepo
}

// New 创建 File UseCase。
func New(objects repo.ObjectStore, fileRepo repo.FileRepo) usecase.File {
	return &useCase{objects: objects, fileRepo: fileRepo}
}

// ObjectStorage 对象存储（MinIO）。

func (u *useCase) GenerateUploadURL(ctx context.Context, key string, expires time.Duration, contentType string) (string, error) {
	url, err := u.objects.PresignUpload(ctx, key, expires, contentType)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorage, err)
	}
	return url, nil
}

func (u *useCase) GetFileURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	url, err := u.objects.PresignDownload(ctx, key, expires)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrStorage, err)
	}
	return url, nil
}

func (u *useCase) DeleteObject(ctx context.Context, key string) error {
	if err := u.objects.Delete(ctx, key); err != nil {
		return fmt.Errorf("%w: %v", ErrStorage, err)
	}
	return nil
}

// FileMetadata 文件元数据（DB）。

func (u *useCase) SaveMeta(ctx context.Context, params input.SaveFileMeta) (int64, error) {
	ef := entity.File{
		ObjectKey:  params.ObjectKey,
		FileName:   params.FileName,
		FileSize:   params.FileSize,
		MimeType:   params.MimeType,
		Usage:      params.Usage,
		ResourceID: params.ResourceID,
		UploaderID: params.UploaderID,
	}
	id, err := u.fileRepo.Create(ctx, ef)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return id, nil
}

func (u *useCase) GetMeta(ctx context.Context, objectKey string) (*output.FileDetail, error) {
	ef, err := u.fileRepo.GetByObjectKey(ctx, objectKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return toFileDetail(ef), nil
}

func (u *useCase) ListFiles(ctx context.Context, params input.ListFiles) (*output.ListResult[output.FileDetail], error) {
	offset := (params.Page - 1) * params.PageSize
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}
	files, total, err := u.fileRepo.List(ctx, offset, params.PageSize, params.Usage, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	items := make([]output.FileDetail, len(files))
	for i, ef := range files {
		items[i] = *toFileDetail(ef)
	}
	return &output.ListResult[output.FileDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) ListMetaByResource(ctx context.Context, usage string, resourceID int64) ([]*output.FileDetail, error) {
	files, err := u.fileRepo.ListByResource(ctx, usage, resourceID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	result := make([]*output.FileDetail, len(files))
	for i, ef := range files {
		result[i] = toFileDetail(ef)
	}
	return result, nil
}

func (u *useCase) BindResource(ctx context.Context, objectKey string, resourceID int64) error {
	if err := u.fileRepo.UpdateResourceID(ctx, objectKey, resourceID); err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return nil
}

func (u *useCase) ClearResourceByUsage(ctx context.Context, usage string, resourceID int64) error {
	if err := u.fileRepo.ClearResourceIDByResourceAndUsage(ctx, resourceID, usage); err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return nil
}

func (u *useCase) DeleteWithMeta(ctx context.Context, objectKey string) error {
	if err := u.objects.Delete(ctx, objectKey); err != nil {
		return fmt.Errorf("%w: %v", ErrStorage, err)
	}
	if err := u.fileRepo.DeleteByObjectKey(ctx, objectKey); err != nil {
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return nil
}

func toFileDetail(ef *entity.File) *output.FileDetail {
	return &output.FileDetail{
		ID:         ef.ID,
		ObjectKey:  ef.ObjectKey,
		FileName:   ef.FileName,
		FileSize:   ef.FileSize,
		MimeType:   ef.MimeType,
		Usage:      ef.Usage,
		ResourceID: ef.ResourceID,
		UploaderID: ef.UploaderID,
		CreatedAt:  ef.CreatedAt,
	}
}
