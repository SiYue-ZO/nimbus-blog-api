package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	"github.com/scc749/nimbus-blog-api/internal/usecase/email"
)

// @Summary 发送邮箱验证码
// @Tags V1.Email
// @Accept json
// @Produce json
// @Param body body request.SendCode true "邮箱与图形验证码"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 502 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/email/send-code [post]
func (r *V1) sendEmailCode(ctx fiber.Ctx) error {
	if r.email == nil || r.captcha == nil {
		// usecase not injected; treat as configuration/bootstrap error
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorConfigNotLoaded, "service not initialized")
	}

	var body request.SendCode
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - email - sendCode - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - v1 - email - sendCode - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	// 验证图形验证码，成功后再发送邮件验证码
	ok, err := r.captcha.Verify(ctx.Context(), body.CaptchaID, body.Captcha)
	if err != nil {
		r.logger.Error(err, "http - v1 - email - sendCode - captcha verify")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "captcha verification failed")
	}
	if !ok {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorVerificationCode, "invalid captcha")
	}

	if err := r.email.SendCode(ctx.Context(), body.Email); err != nil {
		r.logger.Error(err, "http - v1 - email - sendCode - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "failed to send verification code"

		switch {
		case errors.Is(err, email.ErrCodeGeneration):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorSystem
			msg = "failed to generate verification code"
		case errors.Is(err, email.ErrEmailSend):
			httpCode = http.StatusBadGateway
			bizCode = response.ErrorThirdParty
			msg = "failed to send email"
		case errors.Is(err, email.ErrCodeStore):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorSystem
			msg = "failed to store verification code"
		}

		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}

	return sharedresp.WriteSuccess(ctx)
}
