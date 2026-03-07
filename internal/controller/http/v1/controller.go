package v1

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	authuser "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	"github.com/scc749/nimbus-blog-api/pkg/logger"
)

type V1 struct {
	logger       logger.Interface
	validate     *validator.Validate
	captcha      usecase.Captcha
	email        usecase.Email
	auth         usecase.UserAuth
	file         usecase.File
	user         usecase.User
	content      usecase.Content
	comment      usecase.Comment
	feedback     usecase.Feedback
	link         usecase.Link
	setting      usecase.Setting
	notification usecase.Notification
	signer       authuser.TokenSigner
}

func optionalUserID(ctx fiber.Ctx) *int64 {
	claims, ok := authuser.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return nil
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return nil
	}
	return &uid
}
