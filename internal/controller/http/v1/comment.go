package v1

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	commentUC "github.com/scc749/nimbus-blog-api/internal/usecase/comment"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 文章评论列表
// @Description 可选携带 BearerAuth，以返回点赞状态（like.liked）。
// @Tags V1.Comments
// @Produce json
// @Param id path int true "文章 ID"
// @Success 200 {object} sharedresp.Envelope{data=[]response.CommentBasic}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/posts/{id}/comments [get]
func (r *V1) listComments(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing post id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid post id")
	}

	userID := optionalUserID(ctx)

	result, err := r.comment.GetAllPublicCommentsByPostID(ctx.Context(), nid, userID)
	if err != nil {
		r.logger.Error(err, "http - v1 - comment - listComments - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListCommentsFailed, "failed to list comments")
	}

	list := make([]response.CommentBasic, 0, len(result.Items))
	for _, c := range result.Items {
		list = append(list, response.CommentBasic{
			ID:           c.ID,
			PostID:       c.PostID,
			ParentID:     c.ParentID,
			UserID:       c.UserID,
			Content:      c.Content,
			Like:         response.LikeInfo{Liked: c.Like.Liked, Likes: c.Like.Likes},
			RepliesCount: c.RepliesCount,
			CreatedAt:    c.CreatedAt,
			UserProfile: response.CommentUserProfile{
				Name:    c.UserProfile.Name,
				Avatar:  c.UserProfile.Avatar,
				Bio:     c.UserProfile.Bio,
				Status:  c.UserProfile.Status,
				BlogURL: c.UserProfile.BlogURL,
				Email:   c.UserProfile.Email,
				Region:  c.UserProfile.Region,
			},
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(list))
}

// @Summary 提交评论
// @Tags V1.Comments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "文章 ID"
// @Param body body request.SubmitComment true "评论内容"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/content/posts/{id}/comments [post]
func (r *V1) submitComment(ctx fiber.Ctx) error {
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
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing post id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid post id")
	}

	var body request.SubmitComment
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - comment - submitComment - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	ip := ctx.IP()
	ua := ctx.Get("User-Agent")
	if err := r.comment.SubmitComment(ctx.Context(), input.SubmitComment{
		PostID:    nid,
		ParentID:  body.ParentID,
		UserID:    uid,
		Content:   body.Content,
		IPAddress: &ip,
		UserAgent: &ua,
	}); err != nil {
		if errors.Is(err, commentUC.ErrInvalidParent) {
			return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorSubmitCommentFailed, "invalid parent comment")
		}
		r.logger.Error(err, "http - v1 - comment - submitComment - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSubmitCommentFailed, "submit comment failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 点赞/取消点赞评论
// @Tags V1.Comments
// @Produce json
// @Security BearerAuth
// @Param id path int true "评论 ID"
// @Success 200 {object} sharedresp.Envelope{data=response.LikeInfo}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/comments/{id}/likes [post]
func (r *V1) toggleCommentLike(ctx fiber.Ctx) error {
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
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing comment id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid comment id")
	}

	liked, likes, err := r.comment.ToggleLikeOnComment(ctx.Context(), nid, uid)
	if err != nil {
		r.logger.Error(err, "http - v1 - comment - toggleCommentLike - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorLikeCommentFailed, "like comment failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.LikeInfo{Liked: &liked, Likes: likes}))
}

// @Summary 取消点赞评论
// @Tags V1.Comments
// @Produce json
// @Security BearerAuth
// @Param id path int true "评论 ID"
// @Success 200 {object} sharedresp.Envelope{data=response.LikeInfo}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/comments/{id}/likes [delete]
func (r *V1) removeCommentLike(ctx fiber.Ctx) error {
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
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing comment id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid comment id")
	}

	liked, likes, err := r.comment.RemoveLikeOnComment(ctx.Context(), nid, uid)
	if err != nil {
		r.logger.Error(err, "http - v1 - comment - removeCommentLike - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUnlikeCommentFailed, "unlike comment failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.LikeInfo{Liked: &liked, Likes: likes}))
}

// @Summary 删除自己的评论
// @Tags V1.Comments
// @Produce json
// @Security BearerAuth
// @Param id path int true "评论 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 403 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/comments/{id} [delete]
func (r *V1) deleteComment(ctx fiber.Ctx) error {
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
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing comment id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid comment id")
	}

	if err := r.comment.DeleteOwnComment(ctx.Context(), nid, uid); err != nil {
		if errors.Is(err, commentUC.ErrForbidden) {
			return sharedresp.WriteError(ctx, http.StatusForbidden, response.ErrorPermissionDenied, "cannot delete others comment")
		}
		r.logger.Error(err, "http - v1 - comment - deleteComment - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeleteCommentFailed, "delete comment failed")
	}
	return sharedresp.WriteSuccess(ctx)
}
