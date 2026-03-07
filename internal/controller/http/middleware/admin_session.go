package middleware

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	adminresp "github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
)

func NewAdminSessionMiddleware(store *session.Store) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return sharedresp.WriteError(ctx, http.StatusUnauthorized, adminresp.ErrorAdminSessionMissing, "unauthorized")
		}
		if sess == nil {
			return sharedresp.WriteError(ctx, http.StatusUnauthorized, adminresp.ErrorAdminSessionMissing, "unauthorized")
		}
		if id := sess.Get("admin_id"); id == nil {
			return sharedresp.WriteError(ctx, http.StatusUnauthorized, adminresp.ErrorAdminSessionMissing, "unauthorized")
		}
		ctx.Locals("admin_id", sess.Get("admin_id"))
		return ctx.Next()
	}
}
