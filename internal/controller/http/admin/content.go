package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	contentUC "github.com/scc749/nimbus-blog-api/internal/usecase/content"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 文章列表（管理端）
// @Tags Admin.Content
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param keyword query string false "关键字"
// @Param sort_by query string false "排序字段" Enums(created_at,updated_at,views,likes)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Param filter.category_id query string false "分类 ID 过滤"
// @Param filter.tag_id query string false "标签 ID 过滤"
// @Param filter.status query string false "状态过滤"
// @Param filter.is_featured query string false "是否精选过滤"
// @Success 200 {object} sharedresp.Envelope{data=response.PostSummaryPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/posts [get]
func (r *Admin) listPosts(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("created_at", "updated_at", "views", "likes"), sharedresp.WithAllowedFilters("category_id", "tag_id", "status", "is_featured"))
	pageParams := input.PageParams{
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}
	var keywordParams *input.KeywordParams
	if pq.Keyword != "" {
		keywordParams = &input.KeywordParams{
			Keyword: pq.Keyword,
		}
	}
	var sortParams *input.SortParams
	if pq.SortBy != "" {
		sortParams = &input.SortParams{
			SortBy: pq.SortBy,
			Order:  pq.Order,
		}
	}
	var categoryID input.IntFilterParam
	if c, ok := pq.Filters["category_id"]; ok && c != "" {
		categoryID = input.ParseIntFilterParam(c)
	}
	var tagID input.IntFilterParam
	if t, ok := pq.Filters["tag_id"]; ok && t != "" {
		tagID = input.ParseIntFilterParam(t)
	}
	var status input.StringFilterParam
	if s, ok := pq.Filters["status"]; ok && s != "" {
		status = input.ParseStringFilterParam(s)
	}
	var isFeatured input.BoolFilterParam
	if f, ok := pq.Filters["is_featured"]; ok && f != "" {
		isFeatured = input.ParseBoolFilterParam(f)
	}
	result, err := r.content.ListPosts(ctx.Context(), input.ListPosts{
		PageParams: pageParams,
		Keyword:    keywordParams,
		Sort:       sortParams,
		Status:     status,
		CategoryID: categoryID,
		TagID:      tagID,
		IsFeatured: isFeatured,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - content - listPosts - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "failed to list posts")
	}
	list := make([]response.PostSummary, 0, len(result.Items))
	for _, p := range result.Items {
		author := response.AuthorInfo{ID: p.Author.ID, Nickname: p.Author.Nickname, Specialization: p.Author.Specialization}
		category := response.BaseCategory{
			ID:   p.Category.ID,
			Name: p.Category.Name,
			Slug: p.Category.Slug,
		}
		tags := make([]response.BaseTag, 0, len(p.Tags))
		for _, t := range p.Tags {
			tags = append(tags, response.BaseTag{
				ID:   t.ID,
				Name: t.Name,
				Slug: t.Slug,
			})
		}
		list = append(list, response.PostSummary{
			ID:            p.ID,
			Title:         p.Title,
			Slug:          p.Slug,
			Excerpt:       p.Excerpt,
			FeaturedImage: p.FeaturedImage,
			AuthorID:      p.AuthorID,
			Author:        author,
			Status:        p.Status,
			ReadTime:      p.ReadTime,
			Views:         p.Views,
			Likes:         p.Like.Likes,
			IsFeatured:    p.IsFeatured,
			PublishedAt:   p.PublishedAt,
			CreatedAt:     p.CreatedAt,
			UpdatedAt:     p.UpdatedAt,
			Category:      category,
			Tags:          tags,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 文章详情（按 ID）
// @Tags Admin.Content
// @Produce json
// @Security AdminSession
// @Param id path int true "文章 ID"
// @Success 200 {object} sharedresp.Envelope{data=response.PostDetail}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/posts/{id} [get]
func (r *Admin) getPost(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	post, err := r.content.GetPostByID(ctx.Context(), nid)
	if err != nil {
		r.logger.Error(err, "http - admin - content - getPost - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "failed to get post"
		switch {
		case errors.Is(err, contentUC.ErrNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorPostNotFound
			msg = "post not found"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	category := response.BaseCategory{
		ID:   post.Category.ID,
		Name: post.Category.Name,
		Slug: post.Category.Slug,
	}
	tags := make([]response.BaseTag, 0, len(post.Tags))
	for _, t := range post.Tags {
		tags = append(tags, response.BaseTag{
			ID:   t.ID,
			Name: t.Name,
			Slug: t.Slug,
		})
	}
	author := response.AuthorInfo{ID: post.Author.ID, Nickname: post.Author.Nickname, Specialization: post.Author.Specialization}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(
		response.PostDetail{
			ID:              post.ID,
			Title:           post.Title,
			Slug:            post.Slug,
			Excerpt:         post.Excerpt,
			FeaturedImage:   post.FeaturedImage,
			AuthorID:        post.AuthorID,
			Author:          author,
			Status:          post.Status,
			ReadTime:        post.ReadTime,
			Views:           post.Views,
			Likes:           post.Like.Likes,
			IsFeatured:      post.IsFeatured,
			PublishedAt:     post.PublishedAt,
			CreatedAt:       post.CreatedAt,
			UpdatedAt:       post.UpdatedAt,
			Category:        category,
			Tags:            tags,
			Content:         post.Content,
			MetaTitle:       post.MetaTitle,
			MetaDescription: post.MetaDescription,
		},
	))
}

// @Summary 创建文章
// @Tags Admin.Content
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.CreatePost true "文章内容"
// @Success 200 {object} sharedresp.Envelope{data=response.CreatePost}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/posts [post]
func (r *Admin) createPost(ctx fiber.Ctx) error {
	var body request.CreatePost
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - content - createPost - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - content - createPost - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if idVal := ctx.Locals("admin_id"); idVal != nil {
		if idStr, ok := idVal.(string); ok && idStr != "" {
			if aid, err := strconv.ParseInt(idStr, 10, 64); err == nil && aid > 0 {
				body.AuthorID = aid
			}
		}
	}
	id, err := r.content.CreatePost(ctx.Context(), input.CreatePost{
		Title:         body.Title,
		Slug:          body.Slug,
		Excerpt:       body.Excerpt,
		Content:       body.Content,
		FeaturedImage: body.FeaturedImage,
		AuthorID:      body.AuthorID,
		CategoryID:    body.CategoryID,
		TagIDs:        body.TagIDs,
		Status:        body.Status,
		IsFeatured:    body.IsFeatured,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - content - createPost - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorCreatePostFailed, "create post failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.CreatePost{ID: id}))
}

// @Summary 更新文章
// @Tags Admin.Content
// @Accept json
// @Produce json
// @Security AdminSession
// @Param id path int true "文章 ID"
// @Param body body request.UpdatePost true "文章内容"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/posts/{id} [put]
func (r *Admin) updatePost(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	var body request.UpdatePost
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - content - updatePost - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// unify: populate body.ID from path before validation
	body.ID = nid
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - content - updatePost - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if body.FeaturedImage != nil && strings.TrimSpace(*body.FeaturedImage) == "" {
		body.FeaturedImage = nil
	}
	if err := r.content.UpdatePost(ctx.Context(), input.UpdatePost{
		ID:            nid,
		Title:         body.Title,
		Slug:          body.Slug,
		Excerpt:       body.Excerpt,
		Content:       body.Content,
		FeaturedImage: body.FeaturedImage,
		AuthorID:      body.AuthorID,
		CategoryID:    body.CategoryID,
		TagIDs:        body.TagIDs,
		Status:        body.Status,
		IsFeatured:    body.IsFeatured,
	}); err != nil {
		r.logger.Error(err, "http - admin - content - updatePost - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpdatePostFailed, "update post failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 删除文章
// @Tags Admin.Content
// @Produce json
// @Security AdminSession
// @Param id path int true "文章 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/posts/{id} [delete]
func (r *Admin) deletePost(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	if err := r.content.DeletePost(ctx.Context(), nid); err != nil {
		r.logger.Error(err, "http - admin - content - deletePost - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeletePostFailed, "delete post failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 分类列表（管理端）
// @Tags Admin.Content
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param keyword query string false "关键字"
// @Param sort_by query string false "排序字段" Enums(name,created_at)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Success 200 {object} sharedresp.Envelope{data=response.CategoryDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/categories [get]
func (r *Admin) listCategories(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("name", "created_at"))
	pageParams := input.PageParams{
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}
	var keywordParams *input.KeywordParams
	if pq.Keyword != "" {
		keywordParams = &input.KeywordParams{
			Keyword: pq.Keyword,
		}
	}
	var sortParams *input.SortParams
	if pq.SortBy != "" {
		sortParams = &input.SortParams{
			SortBy: pq.SortBy,
			Order:  pq.Order,
		}
	}
	result, err := r.content.ListCategories(ctx.Context(), input.ListCategories{
		PageParams: pageParams,
		Keyword:    keywordParams,
		Sort:       sortParams,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - content - listCategories - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "failed to list categories")
	}
	list := make([]response.CategoryDetail, 0, len(result.Items))
	for _, c := range result.Items {
		list = append(list, response.CategoryDetail{
			ID:        c.ID,
			Name:      c.Name,
			Slug:      c.Slug,
			PostCount: c.PostCount,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 创建分类
// @Tags Admin.Content
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.CreateCategory true "分类信息"
// @Success 200 {object} sharedresp.Envelope{data=response.CreateCategory}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/categories [post]
func (r *Admin) createCategory(ctx fiber.Ctx) error {
	var body request.CreateCategory
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - content - createCategory - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - content - createCategory - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	id, err := r.content.CreateCategory(ctx.Context(), input.CreateCategory{
		Name: body.Name,
		Slug: body.Slug,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - content - createCategory - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorCreateCategoryFailed, "create category failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.CreateCategory{ID: id}))
}

// @Summary 更新分类
// @Tags Admin.Content
// @Accept json
// @Produce json
// @Security AdminSession
// @Param id path int true "分类 ID"
// @Param body body request.UpdateCategory true "分类信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/categories/{id} [put]
func (r *Admin) updateCategory(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	var body request.UpdateCategory
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - content - updateCategory - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// unify: populate body.ID from path before validation
	body.ID = nid
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - content - updateCategory - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.content.UpdateCategory(ctx.Context(), input.UpdateCategory{
		ID:   nid,
		Name: body.Name,
		Slug: body.Slug,
	}); err != nil {
		r.logger.Error(err, "http - admin - content - updateCategory - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpdateCategoryFailed, "update category failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 删除分类
// @Tags Admin.Content
// @Produce json
// @Security AdminSession
// @Param id path int true "分类 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/categories/{id} [delete]
func (r *Admin) deleteCategory(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	if err := r.content.DeleteCategory(ctx.Context(), nid); err != nil {
		r.logger.Error(err, "http - admin - content - deleteCategory - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeleteCategoryFailed, "delete category failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 标签列表（管理端）
// @Tags Admin.Content
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param keyword query string false "关键字"
// @Param sort_by query string false "排序字段" Enums(name,created_at)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Success 200 {object} sharedresp.Envelope{data=response.TagDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/tags [get]
func (r *Admin) listTags(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("name", "created_at"))
	pageParams := input.PageParams{
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}
	var keywordParams *input.KeywordParams
	if pq.Keyword != "" {
		keywordParams = &input.KeywordParams{
			Keyword: pq.Keyword,
		}
	}
	var sortParams *input.SortParams
	if pq.SortBy != "" {
		sortParams = &input.SortParams{
			SortBy: pq.SortBy,
			Order:  pq.Order,
		}
	}
	result, err := r.content.ListTags(ctx.Context(), input.ListTags{
		PageParams: pageParams,
		Keyword:    keywordParams,
		Sort:       sortParams,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - content - listTags - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "failed to list tags")
	}
	list := make([]response.TagDetail, 0, len(result.Items))
	for _, t := range result.Items {
		list = append(list, response.TagDetail{
			ID:        t.ID,
			Name:      t.Name,
			Slug:      t.Slug,
			PostCount: t.PostCount,
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 创建标签
// @Tags Admin.Content
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.CreateTag true "标签信息"
// @Success 200 {object} sharedresp.Envelope{data=response.CreateTag}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/tags [post]
func (r *Admin) createTag(ctx fiber.Ctx) error {
	var body request.CreateTag
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - content - createTag - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - content - createTag - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	id, err := r.content.CreateTag(ctx.Context(), input.CreateTag{
		Name: body.Name,
		Slug: body.Slug,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - content - createTag - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorCreateTagFailed, "create tag failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.CreateTag{ID: id}))
}

// @Summary 更新标签
// @Tags Admin.Content
// @Accept json
// @Produce json
// @Security AdminSession
// @Param id path int true "标签 ID"
// @Param body body request.UpdateTag true "标签信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/tags/{id} [put]
func (r *Admin) updateTag(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	var body request.UpdateTag
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - content - updateTag - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// unify: populate body.ID from path before validation
	body.ID = nid
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - content - updateTag - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.content.UpdateTag(ctx.Context(), input.UpdateTag{
		ID:   nid,
		Name: body.Name,
		Slug: body.Slug,
	}); err != nil {
		r.logger.Error(err, "http - admin - content - updateTag - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpdateTagFailed, "update tag failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 删除标签
// @Tags Admin.Content
// @Produce json
// @Security AdminSession
// @Param id path int true "标签 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/tags/{id} [delete]
func (r *Admin) deleteTag(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	if err := r.content.DeleteTag(ctx.Context(), nid); err != nil {
		r.logger.Error(err, "http - admin - content - deleteTag - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeleteTagFailed, "delete tag failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 生成 slug
// @Tags Admin.Content
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.GenerateSlug true "输入文本"
// @Success 200 {object} sharedresp.Envelope{data=response.GenerateSlug}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 422 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/content/generate-slug [post]
func (r *Admin) generateSlug(ctx fiber.Ctx) error {
	var body request.GenerateSlug
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - content - generateSlug - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - content - generateSlug - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	slug, err := r.content.GenerateSlug(ctx.Context(), body.Input)
	if err != nil {
		r.logger.Error(err, "http - admin - content - generateSlug - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorGenerateSlugFailed
		msg := "generate slug failed"
		switch {
		case errors.Is(err, contentUC.ErrSlugGenerate):
			httpCode = http.StatusUnprocessableEntity
			msg = "unable to generate slug"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.GenerateSlug{Slug: slug}))
}
