package admin

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 用户列表
// @Tags Admin.Users
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param keyword query string false "关键字"
// @Param sort_by query string false "排序字段" Enums(created_at)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Param filter.status query string false "状态过滤"
// @Success 200 {object} sharedresp.Envelope{data=response.UserDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/users/ [get]
func (r *Admin) listUsers(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("created_at"), sharedresp.WithAllowedFilters("status"))
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
	var status input.StringFilterParam
	if s, ok := pq.Filters["status"]; ok && s != "" {
		status = input.ParseStringFilterParam(s)
	}
	result, err := r.user.ListUsers(ctx.Context(), input.ListUsers{
		PageParams: pageParams,
		Keyword:    keywordParams,
		Sort:       sortParams,
		Status:     status,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - user - listUsers - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListUsersFailed, "failed to list users")
	}
	list := make([]response.UserDetail, 0, len(result.Items))
	for _, u := range result.Items {
		list = append(list, response.UserDetail{
			ID:              u.ID,
			Name:            u.Name,
			Email:           u.Email,
			Avatar:          u.Avatar,
			Bio:             u.Bio,
			Status:          u.Status,
			EmailVerified:   u.EmailVerified,
			Region:          u.Region,
			BlogURL:         u.BlogURL,
			AuthProvider:    u.AuthProvider,
			AuthOpenid:      u.AuthOpenid,
			ShowFullProfile: u.ShowFullProfile,
			CreatedAt:       u.CreatedAt,
			UpdatedAt:       u.UpdatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 更新用户状态
// @Tags Admin.Users
// @Accept json
// @Produce json
// @Security AdminSession
// @Param id path int true "用户 ID"
// @Param body body request.UpdateUserStatus true "状态信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/users/{id}/status [put]
func (r *Admin) updateUserStatus(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	var body request.UpdateUserStatus
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - user - updateUserStatus - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// unify: populate body.ID from path before validation
	body.ID = nid
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.user.UpdateStatus(ctx.Context(), nid, body.Status); err != nil {
		r.logger.Error(err, "http - admin - user - updateUserStatus - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpdateUserStatusFailed, "update user status failed")
	}
	if body.Status == entity.UserStatusDisabled && r.userAuth != nil {
		if err := r.userAuth.RevokeUserRefreshToken(ctx.Context(), nid); err != nil {
			r.logger.Error(err, "http - admin - user - updateUserStatus - revoke refresh token")
		}
	}
	return sharedresp.WriteSuccess(ctx)
}
