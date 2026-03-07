package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
)

// @Summary 站点设置（公开）
// @Tags V1.Settings
// @Produce json
// @Success 200 {object} sharedresp.Envelope{data=[]response.SiteSettingDetail}
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/settings [get]
func (r *V1) listSettings(ctx fiber.Ctx) error {
	result, err := r.setting.GetAllSiteSettings(ctx.Context())
	if err != nil {
		r.logger.Error(err, "http - v1 - setting - listSettings - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorListSettingsFailed, "failed to list settings")
	}
	list := make([]response.SiteSettingDetail, 0)
	for _, s := range result.Items {
		if !s.IsPublic {
			continue
		}
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
