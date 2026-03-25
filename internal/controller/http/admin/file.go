package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

var (
	errUnsupportedUploadType  = errors.New("unsupported upload_type")
	errUnsupportedContentType = errors.New("unsupported content_type")
)

type keyFunc func(body request.GenerateUploadURL) (string, error)

const (
	UploadTypeAvatar      = entity.FileUsageAvatar
	UploadTypePostCover   = entity.FileUsagePostCover
	UploadTypePostContent = entity.FileUsagePostContent
)

// selectKeyFunc 根据 upload_type 返回 object key 生成函数。
// 路径规则：{分类}/{uuid}.ext，不嵌入 resource_id。
// 资源绑定关系由 files 表的 resource_id 字段管理。
func selectKeyFunc(uploadType string) (keyFunc, error) {
	switch uploadType {
	case UploadTypeAvatar:
		return func(body request.GenerateUploadURL) (string, error) {
			ext := extFromContentType(body.ContentType)
			if ext == "" {
				return "", errUnsupportedContentType
			}
			return fmt.Sprintf("avatars/%s%s", uuid.NewString(), ext), nil
		}, nil
	case UploadTypePostCover:
		return func(body request.GenerateUploadURL) (string, error) {
			ext := extFromContentType(body.ContentType)
			if ext == "" {
				return "", errUnsupportedContentType
			}
			return fmt.Sprintf("posts/covers/%s%s", uuid.NewString(), ext), nil
		}, nil
	case UploadTypePostContent:
		return func(body request.GenerateUploadURL) (string, error) {
			ext := extFromContentType(body.ContentType)
			if ext == "" {
				return "", errUnsupportedContentType
			}
			return fmt.Sprintf("posts/content/%s%s", uuid.NewString(), ext), nil
		}, nil
	default:
		return nil, errUnsupportedUploadType
	}
}

func extFromContentType(ct string) string {
	switch ct {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ""
	}
}

// @Summary 生成文件上传 URL
// @Tags Admin.Files
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.GenerateUploadURL true "上传参数"
// @Success 200 {object} sharedresp.Envelope{data=response.FileUploadURL}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 502 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/files/upload-url [post]
func (r *Admin) generateUploadURL(ctx fiber.Ctx) error {
	var body request.GenerateUploadURL
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - file - generateUploadURL - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - file - generateUploadURL - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	f, err := selectKeyFunc(body.UploadType)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorUploadTypeUnsupported, "unsupported upload_type")
	}
	k, err := f(body)
	if err != nil {
		if errors.Is(err, errUnsupportedContentType) {
			return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorContentTypeUnsupported, err.Error())
		}
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, err.Error())
	}

	if body.ExpirySeconds == 0 {
		body.ExpirySeconds = 900
	}

	su, err := r.file.GenerateUploadURL(ctx.Context(), k, time.Duration(body.ExpirySeconds)*time.Second, body.ContentType)
	if err != nil {
		r.logger.Error(err, "http - admin - file - generateUploadURL - usecase")
		return sharedresp.WriteError(ctx, http.StatusBadGateway, response.ErrorGenerateUploadURLFailed, "failed to generate upload url")
	}
	var uploaderID int64
	if idVal := ctx.Locals("admin_id"); idVal != nil {
		if idStr, ok := idVal.(string); ok {
			uploaderID, _ = strconv.ParseInt(idStr, 10, 64)
		}
	}

	var resourceID *int64
	if body.ResourceID > 0 {
		resourceID = &body.ResourceID
	}
	if body.UploadType == UploadTypeAvatar && uploaderID > 0 && resourceID == nil {
		resourceID = &uploaderID
	}

	fileID, err := r.file.SaveMeta(ctx.Context(), input.SaveFileMeta{
		ObjectKey:  k,
		FileName:   body.FileName,
		FileSize:   body.FileSize,
		MimeType:   body.ContentType,
		Usage:      body.UploadType,
		ResourceID: resourceID,
		UploaderID: uploaderID,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - file - generateUploadURL - saveMeta")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSaveFileMetaFailed, "failed to save file metadata")
	}

	dto := response.FileUploadURL{ObjectKey: k, UploadURL: su, Expires: body.ExpirySeconds, FileID: fileID}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(dto))
}

const _fileListURLExpiry = 30 * time.Minute

// @Summary 文件列表
// @Tags Admin.Files
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param sort_by query string false "排序字段" Enums(created_at,file_size)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Param filter.usage query string false "用途过滤"
// @Success 200 {object} sharedresp.Envelope{data=response.FileDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/files/ [get]
func (r *Admin) listFiles(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("created_at", "file_size"), sharedresp.WithAllowedFilters("usage"))
	var sortParams *input.SortParams
	if pq.SortBy != "" {
		sortParams = &input.SortParams{
			SortBy: pq.SortBy,
			Order:  pq.Order,
		}
	}
	var usage input.StringFilterParam
	if s, ok := pq.Filters["usage"]; ok && s != "" {
		usage = input.ParseStringFilterParam(s)
	}
	result, err := r.file.ListFiles(ctx.Context(), input.ListFiles{
		PageParams: input.PageParams{Page: pq.Page, PageSize: pq.PageSize},
		Sort:       sortParams,
		Usage:      usage,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - file - listFiles - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListFilesFailed, "failed to list files")
	}
	list := make([]response.FileDetail, 0, len(result.Items))
	for _, f := range result.Items {
		dl, _ := r.file.GetFileURL(ctx.Context(), f.ObjectKey, _fileListURLExpiry)
		list = append(list, response.FileDetail{
			ID:         f.ID,
			ObjectKey:  f.ObjectKey,
			FileName:   f.FileName,
			FileSize:   f.FileSize,
			MimeType:   f.MimeType,
			Usage:      f.Usage,
			ResourceID: f.ResourceID,
			UploaderID: f.UploaderID,
			URL:        dl,
			CreatedAt:  f.CreatedAt,
		})
	}
	page := sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(page))
}

// @Summary 删除文件
// @Tags Admin.Files
// @Produce json
// @Security AdminSession
// @Param object_key path string true "对象 Key"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 502 {object} sharedresp.Envelope
// @Router /admin/files/{object_key} [delete]
func (r *Admin) deleteFile(ctx fiber.Ctx) error {
	key := ctx.Params("*")
	if key == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing object key")
	}
	if err := r.file.DeleteWithMeta(ctx.Context(), key); err != nil {
		r.logger.Error(err, "http - admin - file - deleteFile - usecase")
		return sharedresp.WriteError(ctx, http.StatusBadGateway, response.ErrorDeleteObjectFailed, "failed to delete object")
	}
	return sharedresp.WriteSuccess(ctx)
}
