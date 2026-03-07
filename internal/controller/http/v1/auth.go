package v1

import (
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/v1/response"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/user"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
)

const RefreshCookieName = "refresh_token"

// @Summary 用户注册
// @Tags V1.Auth
// @Accept json
// @Produce json
// @Param body body request.Register true "注册信息（含邮箱验证码）"
// @Success 200 {object} sharedresp.Envelope{data=response.Register}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/auth/register [post]
func (r *V1) register(ctx fiber.Ctx) error {
	if r.auth == nil || r.email == nil {
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorConfigNotLoaded, "service not initialized")
	}

	var body request.Register
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - auth - register - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - v1 - auth - register - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	// Verify email code before registration
	ok, err := r.email.VerifyCode(ctx.Context(), body.Email, body.Code)
	if err != nil {
		r.logger.Error(err, "http - v1 - auth - register - verify code")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "verification service error")
	}
	if !ok {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorVerificationCode, "invalid verification code")
	}

	user, err := r.auth.Register(ctx.Context(), body.Username, body.Email, body.Password)
	if err != nil {
		r.logger.Error(err, "http - v1 - auth - register - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "registration failed"

		switch {
		case errors.Is(err, authUC.ErrEmailExists):
			httpCode = http.StatusBadRequest
			bizCode = response.ErrorEmailExists
			msg = "email already exists"
		case errors.Is(err, authUC.ErrHashPassword):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorSystem
			msg = "failed to hash password"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}

		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}

	// 自动登录：签发令牌，设置 HttpOnly refresh_token Cookie，并在响应体返回 access_token
	pair, err := r.auth.Login(ctx.Context(), body.Email, body.Password)
	if err != nil {
		r.logger.Error(err, "http - v1 - auth - register - autologin")
		// 注册成功但自动登录失败，不要回滚用户，直接返回登录失败信息
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "auto login failed"
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
		case errors.Is(err, authUC.ErrTokenSign):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorSystem
			msg = "token generation error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}

	ctx.Cookie(&fiber.Cookie{
		Name:     RefreshCookieName,
		Value:    pair.RefreshToken,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Strict",
		Expires:  time.Now().Add(time.Duration(pair.RefreshExpiresIn) * time.Second),
		Path:     "/",
	})

	dto := output.UserDetail{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Avatar:          user.Avatar,
		Bio:             user.Bio,
		Status:          user.Status,
		EmailVerified:   user.EmailVerified,
		Region:          user.Region,
		BlogURL:         user.BlogURL,
		ShowFullProfile: user.ShowFullProfile,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.Register{
		AccessToken: pair.AccessToken,
		TokenType:   pair.TokenType,
		ExpiresIn:   pair.ExpiresIn,
		User:        dto,
	}))
}

// @Summary 用户登录
// @Description 登录成功后通过 HttpOnly Cookie 写入 refresh_token，同时响应体返回 access_token。
// @Tags V1.Auth
// @Accept json
// @Produce json
// @Param body body request.Login true "登录信息（含图形验证码）"
// @Success 200 {object} sharedresp.Envelope{data=response.Login}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 403 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/auth/login [post]
func (r *V1) login(ctx fiber.Ctx) error {
	if r.auth == nil || r.captcha == nil {
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorConfigNotLoaded, "service not initialized")
	}

	var body request.Login
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - auth - login - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - v1 - auth - login - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	// Verify captcha before login
	ok, err := r.captcha.Verify(ctx.Context(), body.CaptchaID, body.Captcha)
	if err != nil {
		r.logger.Error(err, "http - v1 - auth - login - captcha verify")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "captcha verification failed")
	}
	if !ok {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorVerificationCode, "invalid captcha")
	}

	// Perform login
	pair, err := r.auth.Login(ctx.Context(), body.Email, body.Password)
	if err != nil {
		r.logger.Error(err, "http - v1 - auth - login - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "login failed"

		switch {
		case errors.Is(err, authUC.ErrUserNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorUserNotFound
			msg = "user not found"
		case errors.Is(err, authUC.ErrPasswordWrong):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorPasswordWrong
			msg = "wrong password"
		case errors.Is(err, authUC.ErrUserDisabled):
			httpCode = http.StatusForbidden
			bizCode = response.ErrorPermissionDenied
			msg = "account disabled"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		case errors.Is(err, authUC.ErrTokenSign):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorSystem
			msg = "token generation error"
		}

		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}

	// Set HttpOnly refresh token cookie; return only access token in body
	ctx.Cookie(&fiber.Cookie{
		Name:     RefreshCookieName,
		Value:    pair.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  time.Now().Add(time.Duration(pair.RefreshExpiresIn) * time.Second),
		Path:     "/",
	})

	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.Login{
		AccessToken: pair.AccessToken,
		TokenType:   pair.TokenType,
		ExpiresIn:   pair.ExpiresIn,
	}))
}

// @Summary 刷新令牌
// @Description 读取 refresh_token Cookie，返回新的 access_token，并刷新 refresh_token Cookie。
// @Tags V1.Auth
// @Produce json
// @Success 200 {object} sharedresp.Envelope{data=response.Refresh}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 403 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/auth/refresh [post]
func (r *V1) refresh(ctx fiber.Ctx) error {
	if r.auth == nil {
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorConfigNotLoaded, "service not initialized")
	}
	rt := ctx.Cookies(RefreshCookieName)
	if rt == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing refresh token")
	}
	pair, err := r.auth.Refresh(ctx.Context(), rt)
	if err != nil {
		r.logger.Error(err, "http - v1 - auth - refresh - usecase")
		httpCode := http.StatusUnauthorized
		bizCode := response.ErrorTokenInvalid
		msg := "invalid refresh token"
		switch {
		case errors.Is(err, authUC.ErrUserNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorUserNotFound
			msg = "user not found"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		case errors.Is(err, authUC.ErrTokenExpired):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorTokenExpired
			msg = "refresh token expired"
		case errors.Is(err, authUC.ErrTokenInvalid):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorTokenInvalid
			msg = "invalid refresh token"
		case errors.Is(err, authUC.ErrUserDisabled):
			httpCode = http.StatusForbidden
			bizCode = response.ErrorPermissionDenied
			msg = "account disabled"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	ctx.Cookie(&fiber.Cookie{
		Name:     RefreshCookieName,
		Value:    pair.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  time.Now().Add(time.Duration(pair.RefreshExpiresIn) * time.Second),
		Path:     "/",
	})
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.Refresh{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		TokenType:        pair.TokenType,
		ExpiresIn:        pair.ExpiresIn,
		RefreshExpiresIn: pair.RefreshExpiresIn,
	}))
}

// @Summary 忘记密码
// @Tags V1.Auth
// @Accept json
// @Produce json
// @Param body body request.ForgotPassword true "邮箱验证码与新密码"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /v1/auth/forgot [post]
func (r *V1) forgot(ctx fiber.Ctx) error {
	if r.auth == nil || r.email == nil {
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorConfigNotLoaded, "service not initialized")
	}
	var body request.ForgotPassword
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - v1 - auth - forgot - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - v1 - auth - forgot - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	ok, err := r.email.VerifyCode(ctx.Context(), body.Email, body.Code)
	if err != nil {
		r.logger.Error(err, "http - v1 - auth - forgot - verify code")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "verification service error")
	}
	if !ok {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorVerificationCode, "invalid verification code")
	}
	if err := r.auth.ResetPasswordByEmail(ctx.Context(), body.Email, body.NewPassword); err != nil {
		r.logger.Error(err, "http - v1 - auth - forgot - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorSystem
		msg := "reset password failed"
		switch {
		case errors.Is(err, authUC.ErrUserNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorUserNotFound
			msg = "user not found"
		case errors.Is(err, authUC.ErrHashPassword):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorSystem
			msg = "failed to hash password"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 退出登录
// @Description 需要 BearerAuth（由路由层鉴权）；接口会清空 refresh_token Cookie。
// @Tags V1.Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Router /v1/auth/logout [post]
func (r *V1) logout(ctx fiber.Ctx) error {
	if r.auth != nil {
		claims, ok := authUC.AccessClaimsFromContext(ctx.Context())
		if ok && claims != nil {
			uid, err := claims.UserIDInt()
			if err == nil {
				if err := r.auth.RevokeUserRefreshToken(ctx.Context(), uid); err != nil {
					r.logger.Error(err, "http - v1 - auth - logout - revoke refresh token")
					return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorDatabase, "logout failed")
				}
			}
		}
	}
	// Clear refresh token cookie by setting expired cookie
	ctx.Cookie(&fiber.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Path:     "/",
	})
	return sharedresp.WriteSuccess(ctx)
}
