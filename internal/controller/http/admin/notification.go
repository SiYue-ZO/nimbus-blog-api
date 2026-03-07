package admin

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 发送通知
// @Tags Admin.Notifications
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.SendNotification true "通知内容"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/notifications/ [post]
func (r *Admin) sendNotification(ctx fiber.Ctx) error {
	var body request.SendNotification
	if err := ctx.Bind().Body(&body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	if err := r.notify.SendAdminMessage(ctx.Context(), input.SendAdminNotification{
		UserID:  body.UserID,
		Title:   body.Title,
		Content: body.Content,
	}); err != nil {
		r.logger.Error(err, "http - admin - notification - sendNotification - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSendNotificationFailed, "send notification failed")
	}
	return sharedresp.WriteSuccess(ctx)
}
