package comment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
	// ErrForbidden Forbidden 错误哨兵。
	ErrForbidden = errors.New("forbidden")
	// ErrInvalidParent InvalidParent 错误哨兵。
	ErrInvalidParent = errors.New("invalid parent comment")
	// ErrInvalidStatus InvalidStatus 错误哨兵。
	ErrInvalidStatus = errors.New("invalid comment status")
)

type useCase struct {
	comments     repo.CommentRepo
	commentLikes repo.CommentLikeRepo
	users        repo.UserRepo
	posts        repo.PostRepo
	notifier     repo.Notifier
}

// New 创建 Comment UseCase。
func New(comments repo.CommentRepo, commentLikes repo.CommentLikeRepo, users repo.UserRepo, posts repo.PostRepo, notifier repo.Notifier) usecase.Comment {
	return &useCase{comments: comments, commentLikes: commentLikes, users: users, posts: posts, notifier: notifier}
}

// Admin 管理端用例。

func (u *useCase) ListComments(ctx context.Context, params input.ListComments) (*output.ListResult[output.CommentDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var status *string
	if params.Status != nil {
		status = (*string)(params.Status)
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	comments, total, err := u.comments.List(ctx, offset, params.PageSize, status, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	userIDs := collectUniqueUserIDs(comments)
	userMap, err := u.buildUserMap(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	items := make([]output.CommentDetail, len(comments))
	for i, c := range comments {
		items[i] = toCommentDetail(c, userMap)
	}

	return &output.ListResult[output.CommentDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

var validCommentStatuses = map[string]bool{
	entity.CommentStatusApproved: true,
	entity.CommentStatusRejected: true,
	entity.CommentStatusSpam:     true,
}

func (u *useCase) UpdateCommentStatus(ctx context.Context, id int64, status string) error {
	if !validCommentStatuses[status] {
		return fmt.Errorf("%w: %q", ErrInvalidStatus, status)
	}

	c, err := u.comments.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if err := u.comments.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}

	if status != entity.CommentStatusApproved {
		return nil
	}

	post, _ := u.posts.GetByID(ctx, c.PostID)
	var postSlug string
	if post != nil {
		postSlug = post.Slug
	}
	targetURL := ""
	if postSlug != "" {
		targetURL = fmt.Sprintf("/post/%s#comment-%d", postSlug, c.ID)
	}
	meta, _ := json.Marshal(map[string]interface{}{
		entity.NotificationMetaPostID:    c.PostID,
		entity.NotificationMetaPostSlug:  postSlug,
		entity.NotificationMetaCommentID: c.ID,
		entity.NotificationMetaTargetURL: targetURL,
	})

	_ = u.notifier.Send(ctx, entity.Notification{
		UserID:  c.UserID,
		Type:    entity.NotificationTypeCommentApproved,
		Title:   "你的评论已通过审核",
		Content: c.Content,
		Meta:    meta,
	})

	if c.ParentID != nil {
		parent, _ := u.comments.GetByID(ctx, *c.ParentID)
		if parent != nil && parent.UserID != c.UserID {
			metaReply, _ := json.Marshal(map[string]interface{}{
				entity.NotificationMetaPostID:          c.PostID,
				entity.NotificationMetaPostSlug:        postSlug,
				entity.NotificationMetaCommentID:       c.ID,
				entity.NotificationMetaParentCommentID: *c.ParentID,
				entity.NotificationMetaTargetURL:       targetURL,
			})
			_ = u.notifier.Send(ctx, entity.Notification{
				UserID:  parent.UserID,
				Type:    entity.NotificationTypeCommentReply,
				Title:   "你的评论收到了新回复",
				Content: c.Content,
				Meta:    metaReply,
			})
		}
	}

	return nil
}

func (u *useCase) DeleteComment(ctx context.Context, id int64) error {
	if err := u.comments.Delete(ctx, id); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Public 公共端用例。

func (u *useCase) GetAllPublicCommentsByPostID(ctx context.Context, postID int64, userID *int64) (*output.AllResult[output.CommentBasic], error) {
	comments, err := u.comments.ListApprovedByPostID(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	userIDs := collectUniqueUserIDs(comments)
	userMap, err := u.buildUserMap(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	replyCounts := buildReplyCounts(comments)

	items := make([]output.CommentBasic, len(comments))
	for i, c := range comments {
		like := u.getLikeInfo(ctx, c.ID, c.Likes, userID)
		items[i] = toCommentBasic(c, userMap, replyCounts, like)
	}

	return &output.AllResult[output.CommentBasic]{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

func (u *useCase) SubmitComment(ctx context.Context, params input.SubmitComment) error {
	if params.ParentID != nil {
		parent, err := u.comments.GetByID(ctx, *params.ParentID)
		if err != nil {
			return ErrInvalidParent
		}
		if parent.PostID != params.PostID {
			return ErrInvalidParent
		}
		if parent.Status != entity.CommentStatusApproved {
			return ErrInvalidParent
		}
	}

	c := entity.Comment{
		PostID:    params.PostID,
		ParentID:  params.ParentID,
		UserID:    params.UserID,
		Content:   params.Content,
		Status:    entity.CommentStatusPending,
		IPAddress: params.IPAddress,
		UserAgent: params.UserAgent,
	}
	if _, err := u.comments.Create(ctx, c); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func (u *useCase) ToggleLikeOnComment(ctx context.Context, commentID int64, userID int64) (bool, int32, error) {
	liked, count, err := u.commentLikes.Toggle(ctx, commentID, userID)
	if err != nil {
		return false, 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return liked, count, nil
}

func (u *useCase) RemoveLikeOnComment(ctx context.Context, commentID int64, userID int64) (bool, int32, error) {
	removed, count, err := u.commentLikes.Remove(ctx, commentID, userID)
	if err != nil {
		return false, 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return removed, count, nil
}

func (u *useCase) DeleteOwnComment(ctx context.Context, commentID int64, userID int64) error {
	c, err := u.comments.GetByID(ctx, commentID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if c.UserID != userID {
		return ErrForbidden
	}
	if err := u.comments.Delete(ctx, commentID); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

// Helpers 辅助函数。

func collectUniqueUserIDs(comments []*entity.Comment) []int64 {
	seen := make(map[int64]struct{})
	var ids []int64
	for _, c := range comments {
		if _, ok := seen[c.UserID]; !ok {
			seen[c.UserID] = struct{}{}
			ids = append(ids, c.UserID)
		}
	}
	return ids
}

func (u *useCase) buildUserMap(ctx context.Context, userIDs []int64) (map[int64]*entity.User, error) {
	if len(userIDs) == 0 {
		return make(map[int64]*entity.User), nil
	}
	users, err := u.users.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	m := make(map[int64]*entity.User, len(users))
	for _, eu := range users {
		m[eu.ID] = eu
	}
	return m, nil
}

func buildReplyCounts(comments []*entity.Comment) map[int64]int32 {
	counts := make(map[int64]int32)
	for _, c := range comments {
		if c.ParentID != nil {
			counts[*c.ParentID]++
		}
	}
	return counts
}

func toCommentDetail(c *entity.Comment, userMap map[int64]*entity.User) output.CommentDetail {
	d := output.CommentDetail{
		BaseComment: output.BaseComment{
			ID:       c.ID,
			PostID:   c.PostID,
			ParentID: c.ParentID,
			UserID:   c.UserID,
			Content:  c.Content,
		},
		Like:      output.LikeInfo{Likes: c.Likes},
		Status:    c.Status,
		IPAddress: c.IPAddress,
		UserAgent: c.UserAgent,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
	if eu, ok := userMap[c.UserID]; ok {
		d.UserProfile = toUserProfile(eu)
	}
	return d
}

func toCommentBasic(c *entity.Comment, userMap map[int64]*entity.User, replyCounts map[int64]int32, like output.LikeInfo) output.CommentBasic {
	b := output.CommentBasic{
		BaseComment: output.BaseComment{
			ID:           c.ID,
			PostID:       c.PostID,
			ParentID:     c.ParentID,
			UserID:       c.UserID,
			Content:      c.Content,
			RepliesCount: replyCounts[c.ID],
		},
		Like:      like,
		CreatedAt: c.CreatedAt,
	}
	if eu, ok := userMap[c.UserID]; ok {
		b.UserProfile = toUserProfile(eu)
	}
	return b
}

func (u *useCase) getLikeInfo(ctx context.Context, commentID int64, likes int32, userID *int64) output.LikeInfo {
	if userID == nil {
		return output.LikeInfo{Likes: likes}
	}
	liked, err := u.commentLikes.HasLiked(ctx, commentID, *userID)
	if err != nil {
		return output.LikeInfo{Likes: likes}
	}
	return output.LikeInfo{Liked: &liked, Likes: likes}
}

func toUserProfile(eu *entity.User) output.UserProfile {
	p := output.UserProfile{
		Name:    eu.Name,
		Avatar:  eu.Avatar,
		Bio:     eu.Bio,
		Status:  eu.Status,
		BlogURL: eu.BlogURL,
	}
	if eu.ShowFullProfile {
		p.UserProfileExtended = output.UserProfileExtended{
			Email:  eu.Email,
			Region: eu.Region,
		}
	}
	return p
}
