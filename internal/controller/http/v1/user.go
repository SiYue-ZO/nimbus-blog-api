package v1

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	userUC "github.com/scc749/nimbus-blog-api/internal/usecase/user"
)

// @Summary 获取当前用户信息
// @Tags V1.User
// @Produce json
// @Security BearerAuth
// @Success 200 {object} sharedresp.Envelope{data=response.UserProfile}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/user/me [get]
func (r *V1) getMe(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	usr, err := r.user.GetUserByID(ctx.Context(), uid)
	if err != nil {
		r.logger.Error(err, "http - v1 - user - getMe - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "query failed"
		if errors.Is(err, userUC.ErrNotFound) {
			httpCode = http.StatusNotFound
			bizCode = response.ErrorUserNotFound
			msg = "user not found"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}

	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.UserProfile{
		ID:              usr.ID,
		Name:            usr.Name,
		Email:           usr.Email,
		Avatar:          usr.Avatar,
		Bio:             usr.Bio,
		Status:          usr.Status,
		EmailVerified:   usr.EmailVerified,
		Region:          usr.Region,
		BlogURL:         usr.BlogURL,
		ShowFullProfile: usr.ShowFullProfile,
		CreatedAt:       usr.CreatedAt,
		UpdatedAt:       usr.UpdatedAt,
	}))
}

// @Summary 更新个人资料
// @Tags V1.User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body request.UpdateProfile true "资料信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/user/profile [put]
func (r *V1) updateProfile(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	var body request.UpdateProfile
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - user - updateProfile - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	if err := r.user.UpdateProfile(ctx.Context(), uid, input.UpdateProfile{
		Name:            body.Name,
		Bio:             body.Bio,
		Region:          body.Region,
		BlogURL:         body.BlogURL,
		ShowFullProfile: body.ShowFullProfile,
	}); err != nil {
		r.logger.Error(err, "http - v1 - user - updateProfile - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "update failed"
		if errors.Is(err, userUC.ErrNotFound) {
			httpCode = http.StatusNotFound
			bizCode = response.ErrorUserNotFound
			msg = "user not found"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 修改密码
// @Tags V1.User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body request.ChangePassword true "密码信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/user/password [put]
func (r *V1) changePassword(ctx fiber.Ctx) error {
	claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
	if !ok || claims == nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorLoginRequired, "login required")
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorTokenInvalid, "invalid token")
	}

	var body request.ChangePassword
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - user - changePassword - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	if err := r.auth.ChangePassword(ctx.Context(), uid, body.OldPassword, body.NewPassword); err != nil {
		r.logger.Error(err, "http - v1 - user - changePassword - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "change password failed"
		switch {
		case errors.Is(err, authUC.ErrUserNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorUserNotFound
			msg = "user not found"
		case errors.Is(err, authUC.ErrPasswordWrong):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorPasswordWrong
			msg = "wrong password"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	return sharedresp.WriteSuccess(ctx)
}
