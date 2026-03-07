package content

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
)

var (
	// ErrRepo Repo 错误哨兵。
	ErrRepo = errors.New("repo")
	// ErrNotFound NotFound 错误哨兵。
	ErrNotFound = errors.New("not found")
	// ErrSlugGenerate SlugGenerate 错误哨兵。
	ErrSlugGenerate = errors.New("slug generate")
)

type useCase struct {
	translationWebAPI  repo.TranslationWebAPI
	llmWebAPI          repo.LLMWebAPI
	admins             repo.AdminRepo
	posts              repo.PostRepo
	tags               repo.TagRepo
	categories         repo.CategoryRepo
	postLikes          repo.PostLikeRepo
	files              repo.FileRepo
	postViews          repo.PostViewRepo
	readTimeCalculator ReadTimeCalculator
}

// New 创建 Content UseCase。
func New(
	translationWebAPI repo.TranslationWebAPI,
	llmWebAPI repo.LLMWebAPI,
	admins repo.AdminRepo,
	posts repo.PostRepo,
	tags repo.TagRepo,
	categories repo.CategoryRepo,
	postLikes repo.PostLikeRepo,
	files repo.FileRepo,
	postViews repo.PostViewRepo,
	readTimeCalculator ReadTimeCalculator,
) usecase.Content {
	return &useCase{
		translationWebAPI:  translationWebAPI,
		llmWebAPI:          llmWebAPI,
		admins:             admins,
		posts:              posts,
		tags:               tags,
		categories:         categories,
		postLikes:          postLikes,
		files:              files,
		postViews:          postViews,
		readTimeCalculator: readTimeCalculator,
	}
}

// Slug Slug 相关用例。

func (u *useCase) GenerateSlug(ctx context.Context, title string) (string, error) {
	s := strings.TrimSpace(title)
	if s == "" {
		return "", nil
	}
	slugify := func(t string) string {
		lower := strings.ToLower(t)
		re := regexp.MustCompile(`[^a-z0-9]+`)
		slug := re.ReplaceAllString(lower, "-")
		return strings.Trim(slug, "-")
	}
	res, err := u.translationWebAPI.Translate(ctx, s, "auto", "en")
	if err == nil {
		if slug := slugify(res); slug != "" {
			return slug, nil
		}
	}
	msg := fmt.Sprintf("生成slug: [ %s ] → 英文小写连字符，核心关键词", s)
	res, err = u.llmWebAPI.Complete(ctx, "", msg)
	if err == nil {
		lower := strings.ToLower(res)
		re := regexp.MustCompile(`\b[a-z0-9]+(?:-[a-z0-9]+)*\b`)
		if slug := re.FindString(lower); slug != "" {
			return slug, nil
		}
		if slug := slugify(res); slug != "" {
			return slug, nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrSlugGenerate, "slug not generated")
}

// AdminPost 管理端文章用例。

func (u *useCase) ListPosts(ctx context.Context, params input.ListPosts) (*output.ListResult[output.PostSummary], error) {
	offset := (params.Page - 1) * params.PageSize

	var keyword *string
	if params.Keyword != nil {
		keyword = &params.Keyword.Keyword
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}
	var categoryID *int
	if params.CategoryID != nil {
		categoryID = (*int)(params.CategoryID)
	}
	var tagID *int
	if params.TagID != nil {
		tagID = (*int)(params.TagID)
	}
	var status *string
	if params.Status != nil {
		status = (*string)(params.Status)
	}
	var isFeatured *bool
	if params.IsFeatured != nil {
		isFeatured = (*bool)(params.IsFeatured)
	}

	posts, total, err := u.posts.List(ctx, offset, params.PageSize, keyword, sortBy, order, categoryID, tagID, status, isFeatured, false)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items, err := u.toPostSummaries(ctx, posts, nil)
	if err != nil {
		return nil, err
	}

	return &output.ListResult[output.PostSummary]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) GetPostByID(ctx context.Context, id int64) (*output.PostDetail, error) {
	post, err := u.posts.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return u.toPostDetail(ctx, post, nil)
}

func (u *useCase) CreatePost(ctx context.Context, params input.CreatePost) (int64, error) {
	post := entity.Post{
		Title:           params.Title,
		Slug:            params.Slug,
		Excerpt:         params.Excerpt,
		Content:         params.Content,
		FeaturedImage:   params.FeaturedImage,
		AuthorID:        params.AuthorID,
		CategoryID:      params.CategoryID,
		Status:          params.Status,
		IsFeatured:      params.IsFeatured,
		MetaTitle:       &params.Title,
		MetaDescription: params.Excerpt,
	}
	readTime := u.readTimeCalculator.Calculate(params.Content)
	post.ReadTime = &readTime
	if strings.EqualFold(params.Status, entity.PostStatusPublished) {
		now := time.Now()
		post.PublishedAt = &now
	}
	id, err := u.posts.Create(ctx, post)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if len(params.TagIDs) > 0 {
		if err := u.posts.SetTags(ctx, id, params.TagIDs); err != nil {
			return 0, fmt.Errorf("%w: %v", ErrRepo, err)
		}
	}
	u.bindPostFiles(ctx, id, params.FeaturedImage, params.Content)
	return id, nil
}

func (u *useCase) UpdatePost(ctx context.Context, params input.UpdatePost) error {
	post := entity.Post{
		ID:              params.ID,
		Title:           params.Title,
		Slug:            params.Slug,
		Excerpt:         params.Excerpt,
		Content:         params.Content,
		FeaturedImage:   params.FeaturedImage,
		AuthorID:        params.AuthorID,
		CategoryID:      params.CategoryID,
		Status:          params.Status,
		IsFeatured:      params.IsFeatured,
		MetaTitle:       &params.Title,
		MetaDescription: params.Excerpt,
	}
	readTime := u.readTimeCalculator.Calculate(params.Content)
	post.ReadTime = &readTime
	ex, err := u.posts.GetByID(ctx, params.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if ex.PublishedAt == nil && strings.EqualFold(params.Status, entity.PostStatusPublished) {
		now := time.Now()
		post.PublishedAt = &now
	}
	if err := u.posts.Update(ctx, post); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if err := u.posts.SetTags(ctx, params.ID, params.TagIDs); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	u.bindPostFiles(ctx, params.ID, params.FeaturedImage, params.Content)
	return nil
}

func (u *useCase) DeletePost(ctx context.Context, id int64) error {
	if err := u.posts.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// AdminCategory 管理端分类用例。

func (u *useCase) ListCategories(ctx context.Context, params input.ListCategories) (*output.ListResult[output.CategoryDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var keyword *string
	if params.Keyword != nil {
		keyword = &params.Keyword.Keyword
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	categories, total, err := u.categories.List(ctx, offset, params.PageSize, keyword, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items := make([]output.CategoryDetail, len(categories))
	for i, c := range categories {
		items[i] = toCategoryDetail(c)
	}

	return &output.ListResult[output.CategoryDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) CreateCategory(ctx context.Context, params input.CreateCategory) (int64, error) {
	id, err := u.categories.Create(ctx, entity.Category{Name: params.Name, Slug: params.Slug})
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return id, nil
}

func (u *useCase) UpdateCategory(ctx context.Context, params input.UpdateCategory) error {
	if err := u.categories.Update(ctx, entity.Category{ID: params.ID, Name: params.Name, Slug: params.Slug}); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func (u *useCase) DeleteCategory(ctx context.Context, id int64) error {
	if err := u.categories.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// AdminTag 管理端标签用例。

func (u *useCase) ListTags(ctx context.Context, params input.ListTags) (*output.ListResult[output.TagDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var keyword *string
	if params.Keyword != nil {
		keyword = &params.Keyword.Keyword
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	tags, total, err := u.tags.List(ctx, offset, params.PageSize, keyword, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items := make([]output.TagDetail, len(tags))
	for i, t := range tags {
		items[i] = toTagDetail(t)
	}

	return &output.ListResult[output.TagDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) CreateTag(ctx context.Context, params input.CreateTag) (int64, error) {
	id, err := u.tags.Create(ctx, entity.Tag{Name: params.Name, Slug: params.Slug})
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return id, nil
}

func (u *useCase) UpdateTag(ctx context.Context, params input.UpdateTag) error {
	if err := u.tags.Update(ctx, entity.Tag{ID: params.ID, Name: params.Name, Slug: params.Slug}); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func (u *useCase) DeleteTag(ctx context.Context, id int64) error {
	if err := u.tags.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// PublicContent 公共端内容用例。

func (u *useCase) ListPublicPosts(ctx context.Context, params input.ListPublicPosts, userID *int64) (*output.ListResult[output.PostSummary], error) {
	offset := (params.Page - 1) * params.PageSize

	var keyword *string
	if params.Keyword != nil {
		keyword = &params.Keyword.Keyword
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}
	var categoryID *int
	if params.CategoryID != nil {
		categoryID = (*int)(params.CategoryID)
	}
	var tagID *int
	if params.TagID != nil {
		tagID = (*int)(params.TagID)
	}

	published := entity.PostStatusPublished
	featuredFirst := keyword == nil && sortBy == nil && categoryID == nil && tagID == nil

	posts, total, err := u.posts.List(ctx, offset, params.PageSize, keyword, sortBy, order, categoryID, tagID, &published, nil, featuredFirst)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items, err := u.toPostSummaries(ctx, posts, userID)
	if err != nil {
		return nil, err
	}

	return &output.ListResult[output.PostSummary]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) GetPublicPostBySlug(ctx context.Context, slug string, userID *int64) (*output.PostDetail, error) {
	post, err := u.posts.GetBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if !strings.EqualFold(post.Status, entity.PostStatusPublished) {
		return nil, ErrNotFound
	}
	return u.toPostDetail(ctx, post, userID)
}

func (u *useCase) GetAllPublicCategories(ctx context.Context) (*output.AllResult[output.CategoryDetail], error) {
	categories, err := u.categories.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	items := make([]output.CategoryDetail, len(categories))
	for i, c := range categories {
		items[i] = toCategoryDetail(c)
	}
	return &output.AllResult[output.CategoryDetail]{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

func (u *useCase) GetAllPublicTags(ctx context.Context) (*output.AllResult[output.TagDetail], error) {
	tags, err := u.tags.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	items := make([]output.TagDetail, len(tags))
	for i, t := range tags {
		items[i] = toTagDetail(t)
	}
	return &output.AllResult[output.TagDetail]{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

func (u *useCase) ToggleLikeOnPost(ctx context.Context, postID int64, userID int64) (bool, int32, error) {
	liked, count, err := u.postLikes.Toggle(ctx, postID, userID)
	if err != nil {
		return false, 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return liked, count, nil
}

func (u *useCase) RemoveLikeOnPost(ctx context.Context, postID int64, userID int64) (bool, int32, error) {
	removed, count, err := u.postLikes.Remove(ctx, postID, userID)
	if err != nil {
		return false, 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return removed, count, nil
}

func (u *useCase) RecordView(ctx context.Context, postID int64, ip, userAgent, referer string) {
	var ua, ref *string
	if userAgent != "" {
		ua = &userAgent
	}
	if referer != "" {
		ref = &referer
	}
	_ = u.postViews.Record(ctx, entity.PostView{
		PostID:    postID,
		IPAddress: ip,
		UserAgent: ua,
		Referer:   ref,
		ViewedAt:  time.Now(),
	})
}

// FileBinding 文件与资源绑定辅助逻辑。

var reFileURL = regexp.MustCompile(`/api/v1/files/([^\s)"']+)`)

func extractObjectKeys(content string) []string {
	matches := reFileURL.FindAllStringSubmatch(content, -1)
	seen := make(map[string]struct{}, len(matches))
	keys := make([]string, 0, len(matches))
	for _, m := range matches {
		key := m[1]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys
}

func (u *useCase) bindPostFiles(ctx context.Context, postID int64, featuredImage *string, content string) {
	_ = u.files.ClearResourceIDByResourceAndUsage(ctx, postID, entity.FileUsagePostCover)
	_ = u.files.ClearResourceIDByResourceAndUsage(ctx, postID, entity.FileUsagePostContent)

	var keys []string
	if featuredImage != nil && *featuredImage != "" {
		keys = append(keys, *featuredImage)
	}
	keys = append(keys, extractObjectKeys(content)...)
	for _, key := range keys {
		_ = u.files.UpdateResourceID(ctx, key, postID)
	}
}

// Helpers 辅助函数。

func (u *useCase) toPostSummaries(ctx context.Context, posts []*entity.Post, userID *int64) ([]output.PostSummary, error) {
	items := make([]output.PostSummary, len(posts))
	for i, p := range posts {
		author := u.getAuthorInfo(ctx, p.AuthorID)
		like := u.getLikeInfo(ctx, p.ID, p.Likes, userID)
		cat, err := u.categories.GetByID(ctx, p.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRepo, err)
		}
		tags, err := u.tags.ListByPostID(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRepo, err)
		}
		items[i] = toPostSummary(p, author, like, cat, tags)
	}
	return items, nil
}

func (u *useCase) toPostDetail(ctx context.Context, p *entity.Post, userID *int64) (*output.PostDetail, error) {
	author := u.getAuthorInfo(ctx, p.AuthorID)
	like := u.getLikeInfo(ctx, p.ID, p.Likes, userID)
	cat, err := u.categories.GetByID(ctx, p.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	tags, err := u.tags.ListByPostID(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	detail := &output.PostDetail{
		BasePost: toBasePost(p),
		Author:   author,
		Like:     like,
		Category: output.BaseCategory{ID: cat.ID, Name: cat.Name, Slug: cat.Slug},
		Tags:     toBaseTags(tags),
		Content:  p.Content,
	}
	if p.MetaTitle != nil {
		detail.MetaTitle = *p.MetaTitle
	}
	if p.MetaDescription != nil {
		detail.MetaDescription = *p.MetaDescription
	}
	return detail, nil
}

func (u *useCase) getAuthorInfo(ctx context.Context, authorID int64) output.AuthorInfo {
	admin, err := u.admins.GetByID(ctx, authorID)
	if err != nil || admin == nil {
		return output.AuthorInfo{ID: authorID}
	}
	return output.AuthorInfo{ID: admin.ID, Nickname: admin.Nickname, Specialization: admin.Specialization}
}

func (u *useCase) getLikeInfo(ctx context.Context, postID int64, likes int32, userID *int64) output.LikeInfo {
	if userID == nil {
		return output.LikeInfo{Likes: likes}
	}
	liked, err := u.postLikes.HasLiked(ctx, postID, *userID)
	if err != nil {
		return output.LikeInfo{Likes: likes}
	}
	return output.LikeInfo{Liked: &liked, Likes: likes}
}

func toBasePost(p *entity.Post) output.BasePost {
	bp := output.BasePost{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		AuthorID:    p.AuthorID,
		Status:      p.Status,
		Views:       p.Views,
		IsFeatured:  p.IsFeatured,
		PublishedAt: p.PublishedAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
	if p.Excerpt != nil {
		bp.Excerpt = *p.Excerpt
	}
	if p.FeaturedImage != nil {
		bp.FeaturedImage = *p.FeaturedImage
	}
	if p.ReadTime != nil {
		bp.ReadTime = *p.ReadTime
	}
	return bp
}

func toPostSummary(p *entity.Post, author output.AuthorInfo, like output.LikeInfo, cat *entity.Category, tags []*entity.Tag) output.PostSummary {
	return output.PostSummary{
		BasePost: toBasePost(p),
		Author:   author,
		Like:     like,
		Category: output.BaseCategory{ID: cat.ID, Name: cat.Name, Slug: cat.Slug},
		Tags:     toBaseTags(tags),
	}
}

func toBaseTags(tags []*entity.Tag) []output.BaseTag {
	bt := make([]output.BaseTag, len(tags))
	for i, t := range tags {
		bt[i] = output.BaseTag{ID: t.ID, Name: t.Name, Slug: t.Slug}
	}
	return bt
}

func toCategoryDetail(c *entity.Category) output.CategoryDetail {
	return output.CategoryDetail{
		BaseCategory: output.BaseCategory{ID: c.ID, Name: c.Name, Slug: c.Slug},
		PostCount:    c.PostCount,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

func toTagDetail(t *entity.Tag) output.TagDetail {
	return output.TagDetail{
		BaseTag:   output.BaseTag{ID: t.ID, Name: t.Name, Slug: t.Slug},
		PostCount: t.PostCount,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
