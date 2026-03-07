package v1

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 提交反馈
// @Tags V1.Feedbacks
// @Accept json
// @Produce json
// @Param body body request.SubmitFeedback true "反馈内容"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/feedbacks [post]
func (r *V1) submitFeedback(ctx fiber.Ctx) error {
	var body request.SubmitFeedback
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - feedback - submitFeedback - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	feedbackType := body.Type
	if feedbackType == "" {
		feedbackType = entity.FeedbackTypeGeneral
	}
	ip := ctx.IP()
	ua := ctx.Get("User-Agent")

	if err := r.feedback.SubmitFeedback(ctx.Context(), input.SubmitFeedback{
		Name:      body.Name,
		Email:     body.Email,
		Type:      feedbackType,
		Subject:   body.Subject,
		Message:   body.Message,
		IPAddress: &ip,
		UserAgent: &ua,
	}); err != nil {
		r.logger.Error(err, "http - v1 - feedback - submitFeedback - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSubmitFeedbackFailed, "submit feedback failed")
	}
	return sharedresp.WriteSuccess(ctx)
}
