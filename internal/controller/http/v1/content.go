package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	contentUC "github.com/scc749/nimbus-blog-api/internal/usecase/content"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 文章列表
// @Description 可选携带 BearerAuth，以返回点赞状态（like.liked）。
// @Tags V1.Content
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param keyword query string false "关键字"
// @Param filter.category_id query string false "分类 ID 过滤"
// @Param filter.tag_id query string false "标签 ID 过滤"
// @Success 200 {object} sharedresp.Envelope{data=response.PostSummaryPage}
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/posts [get]
func (r *V1) listPosts(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedFilters("category_id", "tag_id"))
	pageParams := input.PageParams{
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}
	var keywordParams *input.KeywordParams
	if pq.Keyword != "" {
		keywordParams = &input.KeywordParams{Keyword: pq.Keyword}
	}
	var sortParams *input.SortParams
	if pq.SortBy != "" {
		sortParams = &input.SortParams{SortBy: pq.SortBy, Order: pq.Order}
	}
	var categoryID input.IntFilterParam
	if c, ok := pq.Filters["category_id"]; ok && c != "" {
		categoryID = input.ParseIntFilterParam(c)
	}
	var tagID input.IntFilterParam
	if t, ok := pq.Filters["tag_id"]; ok && t != "" {
		tagID = input.ParseIntFilterParam(t)
	}

	userID := optionalUserID(ctx)

	result, err := r.content.ListPublicPosts(ctx.Context(), input.ListPublicPosts{
		PageParams: pageParams,
		Keyword:    keywordParams,
		Sort:       sortParams,
		CategoryID: categoryID,
		TagID:      tagID,
	}, userID)
	if err != nil {
		r.logger.Error(err, "http - v1 - content - listPosts - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListPostsFailed, "failed to list posts")
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
			Like:          response.LikeInfo{Liked: p.Like.Liked, Likes: p.Like.Likes},
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

// @Summary 文章详情（按 slug）
// @Description 可选携带 BearerAuth，以返回点赞状态（like.liked）。
// @Tags V1.Content
// @Produce json
// @Param slug path string true "文章 Slug"
// @Success 200 {object} sharedresp.Envelope{data=response.PostDetail}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/posts/{slug} [get]
func (r *V1) getPost(ctx fiber.Ctx) error {
	slug := ctx.Params("slug")
	if slug == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing slug")
	}

	userID := optionalUserID(ctx)

	post, err := r.content.GetPublicPostBySlug(ctx.Context(), slug, userID)
	if err != nil {
		r.logger.Error(err, "http - v1 - content - getPost - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorGetPostFailed
		msg := "failed to get post"
		if errors.Is(err, contentUC.ErrNotFound) {
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
	r.content.RecordView(ctx.Context(), post.ID, ctx.IP(), ctx.Get("User-Agent"), ctx.Get("Referer"))

	author := response.AuthorInfo{ID: post.Author.ID, Nickname: post.Author.Nickname, Specialization: post.Author.Specialization}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.PostDetail{
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
		Like:            response.LikeInfo{Liked: post.Like.Liked, Likes: post.Like.Likes},
		IsFeatured:      post.IsFeatured,
		PublishedAt:     post.PublishedAt,
		CreatedAt:       post.CreatedAt,
		UpdatedAt:       post.UpdatedAt,
		Category:        category,
		Tags:            tags,
		Content:         post.Content,
		MetaTitle:       post.MetaTitle,
		MetaDescription: post.MetaDescription,
	}))
}

// @Summary 分类列表
// @Tags V1.Content
// @Produce json
// @Success 200 {object} sharedresp.Envelope{data=[]response.CategoryDetail}
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/categories [get]
func (r *V1) listCategories(ctx fiber.Ctx) error {
	result, err := r.content.GetAllPublicCategories(ctx.Context())
	if err != nil {
		r.logger.Error(err, "http - v1 - content - listCategories - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListCategoriesFailed, "failed to list categories")
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
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(list))
}

// @Summary 标签列表
// @Tags V1.Content
// @Produce json
// @Success 200 {object} sharedresp.Envelope{data=[]response.TagDetail}
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/tags [get]
func (r *V1) listTags(ctx fiber.Ctx) error {
	result, err := r.content.GetAllPublicTags(ctx.Context())
	if err != nil {
		r.logger.Error(err, "http - v1 - content - listTags - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListTagsFailed, "failed to list tags")
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
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(list))
}

// @Summary 点赞/取消点赞文章
// @Tags V1.Content
// @Produce json
// @Security BearerAuth
// @Param id path int true "文章 ID"
// @Success 200 {object} sharedresp.Envelope{data=response.LikeInfo}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/posts/{id}/likes [post]
func (r *V1) togglePostLike(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}

	liked, likes, err := r.content.ToggleLikeOnPost(ctx.Context(), nid, uid)
	if err != nil {
		r.logger.Error(err, "http - v1 - content - togglePostLike - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorLikePostFailed, "like post failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.LikeInfo{Liked: &liked, Likes: likes}))
}

// @Summary 取消点赞文章
// @Tags V1.Content
// @Produce json
// @Security BearerAuth
// @Param id path int true "文章 ID"
// @Success 200 {object} sharedresp.Envelope{data=response.LikeInfo}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/posts/{id}/likes [delete]
func (r *V1) removePostLike(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}

	liked, likes, err := r.content.RemoveLikeOnPost(ctx.Context(), nid, uid)
	if err != nil {
		r.logger.Error(err, "http - v1 - content - removePostLike - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUnlikePostFailed, "unlike post failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.LikeInfo{Liked: &liked, Likes: likes}))
}
