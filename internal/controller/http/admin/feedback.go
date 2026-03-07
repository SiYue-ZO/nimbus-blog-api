package admin

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 反馈列表
// @Tags Admin.Feedbacks
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param sort_by query string false "排序字段" Enums(created_at)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Param filter.status query string false "状态过滤"
// @Success 200 {object} sharedresp.Envelope{data=response.FeedbackDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/feedbacks/ [get]
func (r *Admin) listFeedbacks(ctx fiber.Ctx) error {
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
	result, err := r.feedback.ListFeedbacks(ctx.Context(), input.ListFeedbacks{
		PageParams: pageParams,
		Sort:       sortParams,
		Status:     status,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - feedback - listFeedbacks - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListFeedbacksFailed, "failed to list feedbacks")
	}
	list := make([]response.FeedbackDetail, 0, len(result.Items))
	for _, f := range result.Items {
		list = append(list, response.FeedbackDetail{
			ID:        f.ID,
			Name:      f.Name,
			Email:     f.Email,
			Type:      f.Type,
			Subject:   f.Subject,
			Message:   f.Message,
			Status:    f.Status,
			IPAddress: f.IPAddress,
			UserAgent: f.UserAgent,
			CreatedAt: f.CreatedAt,
			UpdatedAt: f.UpdatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 反馈详情
// @Tags Admin.Feedbacks
// @Produce json
// @Security AdminSession
// @Param id path int true "反馈 ID"
// @Success 200 {object} sharedresp.Envelope{data=response.FeedbackDetail}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/feedbacks/{id} [get]
func (r *Admin) getFeedback(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	f, err := r.feedback.GetFeedbackByID(ctx.Context(), nid)
	if err != nil {
		r.logger.Error(err, "http - admin - feedback - getFeedback - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorGetFeedbackFailed, "failed to get feedback")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.FeedbackDetail{
		ID:        f.ID,
		Name:      f.Name,
		Email:     f.Email,
		Type:      f.Type,
		Subject:   f.Subject,
		Message:   f.Message,
		Status:    f.Status,
		IPAddress: f.IPAddress,
		UserAgent: f.UserAgent,
		CreatedAt: f.CreatedAt,
		UpdatedAt: f.UpdatedAt,
	}))
}

// @Summary 更新反馈状态
// @Tags Admin.Feedbacks
// @Accept json
// @Produce json
// @Security AdminSession
// @Param id path int true "反馈 ID"
// @Param body body request.UpdateFeedbackStatus true "状态信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/feedbacks/{id}/status [put]
func (r *Admin) updateFeedbackStatus(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	var body request.UpdateFeedbackStatus
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - feedback - updateFeedbackStatus - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// unify: populate body.ID from path before validation
	body.ID = nid
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.feedback.UpdateFeedback(ctx.Context(), input.UpdateFeedback{
		ID:     nid,
		Status: body.Status,
	}); err != nil {
		r.logger.Error(err, "http - admin - feedback - updateFeedbackStatus - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpdateFeedbackStatusFailed, "update feedback status failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 删除反馈
// @Tags Admin.Feedbacks
// @Produce json
// @Security AdminSession
// @Param id path int true "反馈 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/feedbacks/{id} [delete]
func (r *Admin) deleteFeedback(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	if err := r.feedback.DeleteFeedback(ctx.Context(), nid); err != nil {
		r.logger.Error(err, "http - admin - feedback - deleteFeedback - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeleteFeedbackFailed, "delete feedback failed")
	}
	return sharedresp.WriteSuccess(ctx)
}
