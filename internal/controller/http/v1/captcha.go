package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	captchaUC "github.com/scc749/nimbus-blog-api/internal/usecase/captcha"
)

// @Summary 生成验证码
// @Tags V1.Captcha
// @Produce json
// @Success 200 {object} sharedresp.Envelope{data=response.Captcha}
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/captcha/generate [get]
func (r *V1) generateCaptcha(ctx fiber.Ctx) error {
	id, b64s, err := r.captcha.Generate(ctx.Context())
	if err != nil {
		r.logger.Error(err, "http - v1 - captcha - generate - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "failed to generate captcha"
		switch {
		case errors.Is(err, captchaUC.ErrGenerate):
			bizCode = response.ErrorThirdParty
			msg = "failed to generate captcha"
		case errors.Is(err, captchaUC.ErrStore):
			bizCode = response.ErrorCache
			msg = "captcha store error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}

	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.Captcha{
		CaptchaID: id,
		PicPath:   b64s,
	}))
}
