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

// @Summary 友链列表
// @Tags Admin.Links
// @Produce json
// @Security AdminSession
// @Param page query int false "页码" default(1)
// @Param page_size query int false "分页大小" default(10)
// @Param keyword query string false "关键字"
// @Param sort_by query string false "排序字段" Enums(sort_order,created_at,name)
// @Param order query string false "排序方向" Enums(asc,desc) default(desc)
// @Success 200 {object} sharedresp.Envelope{data=response.LinkDetailPage}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/links/ [get]
func (r *Admin) listLinks(ctx fiber.Ctx) error {
	pq := sharedresp.ParsePageQueryWithOptions(ctx, sharedresp.WithAllowedSortBy("sort_order", "created_at", "name"))
	pageParams := input.PageParams{
		Page:     pq.Page,
		PageSize: pq.PageSize,
	}
	listInput := input.ListLinks{
		PageParams: pageParams,
	}
	if pq.Keyword != "" {
		listInput.Keyword = &input.KeywordParams{Keyword: pq.Keyword}
	}
	if pq.SortBy != "" {
		listInput.Sort = &input.SortParams{
			SortBy: pq.SortBy,
			Order:  pq.Order,
		}
	}
	result, err := r.link.ListLinks(ctx.Context(), listInput)
	if err != nil {
		r.logger.Error(err, "http - admin - link - listLinks - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListLinksFailed, "failed to list links")
	}
	list := make([]response.LinkDetail, 0, len(result.Items))
	for _, l := range result.Items {
		list = append(list, response.LinkDetail{
			ID:          l.ID,
			Name:        l.Name,
			URL:         l.URL,
			Description: l.Description,
			Logo:        l.Logo,
			SortOrder:   l.SortOrder,
			Status:      l.Status,
			CreatedAt:   l.CreatedAt,
			UpdatedAt:   l.UpdatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(sharedresp.NewPage(list, result.Page, result.PageSize, result.Total)))
}

// @Summary 创建友链
// @Tags Admin.Links
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.CreateLink true "友链信息"
// @Success 200 {object} sharedresp.Envelope{data=response.CreateLink}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/links/ [post]
func (r *Admin) createLink(ctx fiber.Ctx) error {
	var body request.CreateLink
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - link - createLink - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - link - createLink - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	id, err := r.link.CreateLink(ctx.Context(), input.CreateLink{
		Name:        body.Name,
		URL:         body.URL,
		Description: body.Description,
		Logo:        body.Logo,
		SortOrder:   body.SortOrder,
		Status:      body.Status,
	})
	if err != nil {
		r.logger.Error(err, "http - admin - link - createLink - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorCreateLinkFailed, "create link failed")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.CreateLink{ID: id}))
}

// @Summary 更新友链
// @Tags Admin.Links
// @Accept json
// @Produce json
// @Security AdminSession
// @Param id path int true "友链 ID"
// @Param body body request.UpdateLink true "友链信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/links/{id} [put]
func (r *Admin) updateLink(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	var body request.UpdateLink
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - link - updateLink - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// unify: populate body.ID from path before validation
	body.ID = nid
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - link - updateLink - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.link.UpdateLink(ctx.Context(), input.UpdateLink{
		ID:          nid,
		Name:        body.Name,
		URL:         body.URL,
		Description: body.Description,
		Logo:        body.Logo,
		SortOrder:   body.SortOrder,
		Status:      body.Status,
	}); err != nil {
		r.logger.Error(err, "http - admin - link - updateLink - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpdateLinkFailed, "update link failed")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 删除友链
// @Tags Admin.Links
// @Produce json
// @Security AdminSession
// @Param id path int true "友链 ID"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/links/{id} [delete]
func (r *Admin) deleteLink(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if id == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing id")
	}
	nid, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid id")
	}
	if err := r.link.DeleteLink(ctx.Context(), nid); err != nil {
		r.logger.Error(err, "http - admin - link - deleteLink - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDeleteLinkFailed, "delete link failed")
	}
	return sharedresp.WriteSuccess(ctx)
}
