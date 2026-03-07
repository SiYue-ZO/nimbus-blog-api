package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	commentUC "github.com/scc749/nimbus-blog-api/internal/usecase/comment"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 评论列表
// @Tags Admin.Comments
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param sort_by query string false "排序字段" Enums(created_at)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Param filter.status query string false "状态过滤"
// @Success 200 {object} sharedresp.Envelope{data=response.CommentDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/comments/ [get]
func (r *Admin) listComments(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("created_at"), sharedresp.WithAllowedFilters("status"))
	pageParams := input.PageParams{
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}
	var sortParams *input.SortParams
	if pq.SortBy != "" {
		sortParams = &input.SortParams{
			SortBy: pq.SortBy,
			Order:  pq.Order,
		}
	}
	var status input.StringFilterParam
	if s, ok := pq.Filters["status"]; ok && s != "" {
		status = input.ParseStringFilterParam(s)
	}
	result, err := r.comment.ListComments(ctx.Context(), input.ListComments{
		PageParams: pageParams,
		Sort:       sortParams,
		Status:     status,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - comment - listComments - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListCommentsFailed, "failed to list comments")
	}
	list := make([]response.CommentDetail, 0, len(result.Items))
	for _, c := range result.Items {
		list = append(list, response.CommentDetail{
			ID:           c.ID,
			PostID:       c.PostID,
			ParentID:     c.ParentID,
			UserID:       c.UserID,
			Content:      c.Content,
			Likes:        c.Like.Likes,
			RepliesCount: c.RepliesCount,
			Status:       c.Status,
			IPAddress:    c.IPAddress,
			UserAgent:    c.UserAgent,
			CreatedAt:    c.CreatedAt,
			UpdatedAt:    c.UpdatedAt,
			UserProfile: response.UserProfile{
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
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 更新评论状态
// @Tags Admin.Comments
// @Accept json
// @Produce json
// @Security AdminSession
// @Param id path int true "评论 ID"
// @Param body body request.UpdateCommentStatus true "状态信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/comments/{id}/status [put]
func (r *Admin) updateCommentStatus(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	var body request.UpdateCommentStatus
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - comment - updateCommentStatus - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// unify: populate body.ID from path before validation
	body.ID = nid
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.comment.UpdateCommentStatus(ctx.Context(), nid, body.Status); err != nil {
		r.logger.Error(err, "http - admin - comment - updateCommentStatus - usecase")
		if errors.Is(err, commentUC.ErrInvalidStatus) {
			return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid status")
		}
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpdateCommentStatusFailed, "update comment status failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 删除评论
// @Tags Admin.Comments
// @Produce json
// @Security AdminSession
// @Param id path int true "评论 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/comments/{id} [delete]
func (r *Admin) deleteComment(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	if err := r.comment.DeleteComment(ctx.Context(), nid); err != nil {
		r.logger.Error(err, "http - admin - comment - deleteComment - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeleteCommentFailed, "delete comment failed")
	}
	return sharedresp.WriteSuccess(ctx)
}
