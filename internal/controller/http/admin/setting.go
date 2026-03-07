package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 设置列表
// @Tags Admin.Settings
// @Produce json
// @Security AdminSession
// @Success 200 {object} sharedresp.Envelope{data=[]response.SiteSettingDetail}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/settings/ [get]
func (r *Admin) listSettings(ctx fiber.Ctx) error {
	result, err := r.setting.GetAllSiteSettings(ctx.Context())
	if err != nil {
		r.logger.Error(err, "http - admin - setting - listSettings - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListSettingsFailed, "failed to list settings")
	}
	list := make([]response.SiteSettingDetail, 0, len(result.Items))
	for _, s := range result.Items {
		list = append(list, response.SiteSettingDetail{
			ID:           s.ID,
			SettingKey:   s.SettingKey,
			SettingValue: s.SettingValue,
			SettingType:  s.SettingType,
			Description:  s.Description,
			IsPublic:     s.IsPublic,
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
		})
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(list))
}

// @Summary 获取设置（按 key）
// @Tags Admin.Settings
// @Produce json
// @Security AdminSession
// @Param key path string true "设置 Key"
// @Success 200 {object} sharedresp.Envelope{data=response.SiteSettingDetail}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/settings/{key} [get]
func (r *Admin) getSettingByKey(ctx fiber.Ctx) error {
	key := ctx.Params("key")
	if key == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing key")
	}
	s, err := r.setting.GetSiteSettingByKey(ctx.Context(), key)
	if err != nil {
		r.logger.Error(err, "http - admin - setting - getSettingByKey - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorGetSettingFailed, "failed to get setting")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.SiteSettingDetail{
		ID:           s.ID,
		SettingKey:   s.SettingKey,
		SettingValue: s.SettingValue,
		SettingType:  s.SettingType,
		Description:  s.Description,
		IsPublic:     s.IsPublic,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}))
}

// @Summary Upsert 设置（按 key）
// @Tags Admin.Settings
// @Accept json
// @Produce json
// @Security AdminSession
// @Param key path string true "设置 Key"
// @Param body body request.UpsertSiteSetting true "设置内容"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/settings/{key} [put]
func (r *Admin) upsertSetting(ctx fiber.Ctx) error {
	key := ctx.Params("key")
	if key == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing key")
	}
	var body request.UpsertSiteSetting
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - setting - upsertSetting - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - setting - upsertSetting - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.setting.UpsertSiteSetting(ctx.Context(), input.UpsertSiteSetting{
		SettingKey:   key,
		SettingValue: body.SettingValue,
		SettingType:  body.SettingType,
		Description:  body.Description,
		IsPublic:     body.IsPublic,
	}); err != nil {
		r.logger.Error(err, "http - admin - setting - upsertSetting - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorUpsertSettingFailed, "upsert setting failed")
	}
	if key == "profile.avatar" {
		idVal := ctx.Locals("admin_id")
		idStr, _ := idVal.(string)
		if idStr != "" {
			aid, _ := strconv.ParseInt(idStr, 10, 64)
			_ = r.file.ClearResourceByUsage(ctx.Context(), UploadTypeAvatar, aid)
			if body.SettingValue != nil {
				val := *body.SettingValue
				if !(strings.HasPrefix(val, "/") || strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://")) {
					_ = r.file.BindResource(ctx.Context(), val, aid)
				}
			}
		}
	}
	return sharedresp.WriteSuccess(ctx)
}
