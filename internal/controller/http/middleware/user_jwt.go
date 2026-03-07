package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	v1resp "github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
)

const refreshCookieName = "refresh_token"

func NewUserJWTMiddleware(signer authUC.TokenSigner, userAuth usecase.UserAuth) fiber.Handler {
	return func(c fiber.Ctx) error {
		if userAuth == nil {
			return sharedresp.WriteError(c, http.StatusInternalServerError, v1resp.ErrorConfigNotLoaded, "service not initialized")
		}
		authorization := c.Get("Authorization")
		if authorization == "" {
			return sharedresp.WriteError(c, http.StatusUnauthorized, v1resp.ErrorLoginRequired, "login required")
		}
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return sharedresp.WriteError(c, http.StatusBadRequest, v1resp.ErrorParamFormat, "invalid Authorization format")
		}
		tokenStr := parts[1]

		claims, err := signer.ParseAccess(tokenStr)
		if err != nil {
			if errors.Is(err, authUC.ErrTokenExpired) {
				return sharedresp.WriteError(c, http.StatusUnauthorized, v1resp.ErrorTokenExpired, "token expired")
			}
			return sharedresp.WriteError(c, http.StatusUnauthorized, v1resp.ErrorTokenInvalid, "invalid token")
		}

		uid, err := claims.UserIDInt()
		if err != nil {
			return sharedresp.WriteError(c, http.StatusUnauthorized, v1resp.ErrorTokenInvalid, "invalid token")
		}
		rt := c.Cookies(refreshCookieName)
		if rt == "" {
			return sharedresp.WriteError(c, http.StatusUnauthorized, v1resp.ErrorTokenInvalid, "invalid token")
		}
		if err := userAuth.ValidateSession(c.Context(), uid, rt); err != nil {
			if errors.Is(err, authUC.ErrTokenExpired) {
				return sharedresp.WriteError(c, http.StatusUnauthorized, v1resp.ErrorTokenExpired, "token expired")
			}
			if errors.Is(err, authUC.ErrUserDisabled) {
				return sharedresp.WriteError(c, http.StatusForbidden, v1resp.ErrorPermissionDenied, "account disabled")
			}
			if errors.Is(err, authUC.ErrTokenInvalid) {
				return sharedresp.WriteError(c, http.StatusUnauthorized, v1resp.ErrorTokenInvalid, "invalid token")
			}
			return sharedresp.WriteError(c, http.StatusInternalServerError, v1resp.ErrorDatabase, "database error")
		}

		c.Locals("claims", claims)
		c.SetContext(authUC.WithAccessClaims(c.Context(), claims))
		return c.Next()
	}
}

func NewOptionalUserJWTMiddleware(signer authUC.TokenSigner, userAuth usecase.UserAuth) fiber.Handler {
	return func(c fiber.Ctx) error {
		if userAuth == nil {
			return c.Next()
		}
		authorization := c.Get("Authorization")
		if authorization == "" {
			return c.Next()
		}
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return c.Next()
		}
		tokenStr := parts[1]

		claims, err := signer.ParseAccess(tokenStr)
		if err != nil {
			return c.Next()
		}
		uid, err := claims.UserIDInt()
		if err != nil {
			return c.Next()
		}
		rt := c.Cookies(refreshCookieName)
		if rt == "" {
			return c.Next()
		}
		if err := userAuth.ValidateSession(c.Context(), uid, rt); err != nil {
			return c.Next()
		}
		c.Locals("claims", claims)
		c.SetContext(authUC.WithAccessClaims(c.Context(), claims))
		return c.Next()
	}
}
