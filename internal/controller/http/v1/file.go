package v1

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
)

const _defaultFileURLExpiry = 1 * time.Hour

// @Summary 获取文件临时访问 URL（重定向）
// @Tags V1.Files
// @Param object_key path string true "对象 Key"
// @Success 307 {string} string "Temporary Redirect"
// @Failure 400 {object} sharedresp.Envelope
// @Failure 502 {object} sharedresp.Envelope
// @Router /v1/files/{object_key} [get]
func (r *V1) getFileURL(ctx fiber.Ctx) error {
	key := ctx.Params("*")
	if key == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing file key")
	}
	dl, err := r.file.GetFileURL(ctx.Context(), key, _defaultFileURLExpiry)
	if err != nil {
		r.logger.Error(err, "http - v1 - file - getFileURL - usecase")
		return sharedresp.WriteError(ctx, http.StatusBadGateway, response.ErrorGetFileURLFailed, "failed to get file url")
	}
	if u, perr := url.Parse(dl); perr == nil {
		proto := ctx.Get("X-Forwarded-Proto")
		if proto == "" {
			proto = "http"
		}
		host := ctx.Get("X-Forwarded-Host")
		if host == "" {
			host = ctx.Hostname()
		}
		dl = fmt.Sprintf("%s://%s/minio%s", proto, host, u.RequestURI())
	}
	return ctx.Redirect().Status(http.StatusTemporaryRedirect).To(dl)
}
