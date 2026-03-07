package admin

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/request"
	"github.com/scc749/nimbus-blog-api/internal/controller/http/admin/response"
	sharedresp "github.com/scc749/nimbus-blog-api/internal/controller/http/shared"
	authUC "github.com/scc749/nimbus-blog-api/internal/usecase/auth/admin"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
)

// @Summary 管理员登录
// @Description 登录成功后通过 Cookie 写入 AdminSession（fiber_session）。
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Param body body request.Login true "登录信息"
// @Success 200 {object} sharedresp.Envelope{data=response.Login}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/login [post]
func (r *Admin) login(ctx fiber.Ctx) error {
	var body request.Login
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - auth - login - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - auth - login - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}

	admin, err := r.auth.Login(ctx.Context(), body.Username, body.Password)
	if err != nil {
		r.logger.Error(err, "http - admin - auth - login - usecase")
		httpCode := http.StatusUnauthorized
		bizCode := response.ErrorUnauthorized
		msg := "invalid credentials"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrPasswordWrong):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorAdminPasswordWrong
			msg = "wrong password"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}

	// Handle mandatory password reset indication
	if admin.MustResetPassword {
		dto := response.Login{RequiresReset: true, OTPRequired: false}
		return sharedresp.WriteSuccess(ctx, sharedresp.WithData(dto))
	}

	if admin.TwoFactorSecret != nil && *admin.TwoFactorSecret != "" {
		if body.OTPCode != "" {
			ok, verr := r.auth.ValidateTOTP(ctx.Context(), admin.ID, body.OTPCode)
			if verr != nil {
				r.logger.Error(verr, "http - admin - auth - login - otp validate")
				return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "otp validate failed")
			}
			if !ok {
				return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminOTPWrong, "invalid otp code")
			}
		} else if body.RecoveryCode != "" {
			ok, err := r.auth.VerifyAndUseRecoveryCode(ctx.Context(), admin.ID, body.RecoveryCode)
			if err != nil {
				r.logger.Error(err, "http - admin - auth - login - recovery verify")
				return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "recovery verify failed")
			}
			if !ok {
				return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminRecoveryCodeWrong, "invalid recovery code")
			}
		} else {
			dto := response.Login{RequiresReset: false, OTPRequired: true}
			return sharedresp.WriteSuccess(ctx, sharedresp.WithData(dto))
		}
	}

	sess, err := r.sess.Get(ctx)
	if err != nil {
		r.logger.Error(err, "http - admin - auth - login - session get")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorConfigNotLoaded, "failed to init session")
	}
	sess.Set("admin_id", strconv.FormatInt(admin.ID, 10))
	if body.RecoveryCode != "" {
		sess.Set("recovery_login", true)
	}
	// Reasonable default: 24h admin session
	sess.SetIdleTimeout(24 * time.Hour)
	if err := sess.Save(); err != nil {
		r.logger.Error(err, "http - admin - auth - login - session save")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorConfigNotLoaded, "failed to save session")
	}

	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.Login{RequiresReset: false, OTPRequired: false}))
}

// @Summary 管理员重置密码
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Param body body request.ResetPassword true "重置信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/reset [post]
func (r *Admin) resetPassword(ctx fiber.Ctx) error {
	var body request.ResetPassword
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - auth - resetPassword - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - auth - resetPassword - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	// Validate credentials first
	admin, err := r.auth.Login(ctx.Context(), body.Username, body.OldPassword)
	if err != nil {
		httpCode := http.StatusUnauthorized
		bizCode := response.ErrorUnauthorized
		msg := "invalid credentials"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrPasswordWrong):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorAdminPasswordWrong
			msg = "wrong password"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	if err := r.auth.ChangePassword(ctx.Context(), input.ChangePassword{
		ID:             admin.ID,
		OldPassword:    body.OldPassword,
		NewPassword:    body.NewPassword,
		ClearResetFlag: true,
	}); err != nil {
		r.logger.Error(err, "http - admin - auth - resetPassword - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorAdminPasswordChangeFailed
		msg := "failed to change password"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrPasswordWrong):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorAdminPasswordWrong
			msg = "wrong password"
		case errors.Is(err, authUC.ErrHashPassword):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorAdminPasswordChangeFailed
			msg = "hash password error"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 管理员修改密码
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.ChangePassword true "密码信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/password [put]
func (r *Admin) changePassword(ctx fiber.Ctx) error {
	idVal := ctx.Locals("admin_id")
	idStr, _ := idVal.(string)
	if idStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminSessionMissing, "unauthorized")
	}
	var body request.ChangePassword
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - auth - changePassword - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - auth - changePassword - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	aid, _ := strconv.ParseInt(idStr, 10, 64)
	if err := r.auth.ChangePassword(ctx.Context(), input.ChangePassword{
		ID:          aid,
		OldPassword: body.OldPassword,
		NewPassword: body.NewPassword,
	}); err != nil {
		r.logger.Error(err, "http - admin - auth - changePassword - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorAdminPasswordChangeFailed
		msg := "failed to change password"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrPasswordWrong):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorAdminPasswordWrong
			msg = "wrong password"
		case errors.Is(err, authUC.ErrHashPassword):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorAdminPasswordChangeFailed
			msg = "hash password error"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 管理员登出
// @Description 清理 AdminSession Cookie（若存在）。
// @Tags Admin.Auth
// @Produce json
// @Success 200 {object} sharedresp.Envelope
// @Router /admin/auth/logout [post]
func (r *Admin) logout(ctx fiber.Ctx) error {
	sess, err := r.sess.Get(ctx)
	if err == nil && sess != nil {
		_ = sess.Destroy()
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 获取管理员资料
// @Tags Admin.Auth
// @Produce json
// @Security AdminSession
// @Success 200 {object} sharedresp.Envelope{data=response.AdminProfile}
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/profile [get]
func (r *Admin) getProfile(ctx fiber.Ctx) error {
	idVal := ctx.Locals("admin_id")
	idStr, _ := idVal.(string)
	if idStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminSessionMissing, "unauthorized")
	}
	aid, _ := strconv.ParseInt(idStr, 10, 64)
	profile, err := r.auth.GetProfile(ctx.Context(), aid)
	if err != nil {
		r.logger.Error(err, "http - admin - auth - getProfile - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorAdminNotFound
		msg := "admin not found"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
		case errors.Is(err, authUC.ErrRepo):
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	dto := response.AdminProfile{
		Nickname:       profile.Nickname,
		Specialization: profile.Specialization,
		TwoFAEnabled:   profile.TwoFAEnabled,
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(dto))
}

// @Summary 更新管理员资料
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.UpdateProfile true "资料信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/profile [put]
func (r *Admin) updateProfile(ctx fiber.Ctx) error {
	idVal := ctx.Locals("admin_id")
	idStr, _ := idVal.(string)
	if idStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminSessionMissing, "unauthorized")
	}
	var body request.UpdateProfile
	if err := ctx.Bind().Body(&body); err != nil {
		r.logger.Error(err, "http - admin - auth - updateProfile - parse body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		r.logger.Error(err, "http - admin - auth - updateProfile - validate body")
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	aid, _ := strconv.ParseInt(idStr, 10, 64)
	if err := r.auth.UpdateProfile(ctx.Context(), input.UpdateAdminProfile{
		ID:             aid,
		Nickname:       body.Nickname,
		Specialization: body.Specialization,
	}); err != nil {
		r.logger.Error(err, "http - admin - auth - updateProfile - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorAdminUpdateProfileFailed
		msg := "failed to update profile"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 生成/刷新 2FA 配置
// @Description 仅生成密钥与二维码并缓存（返回 setup_id）；不会写入数据库。前端需调用 /2fa/verify 完成启用。
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.TwoFASetup true "空对象"
// @Success 200 {object} sharedresp.Envelope{data=response.TwoFASetupStart}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/2fa/setup [post]
func (r *Admin) twoFASetup(ctx fiber.Ctx) error {
	idVal := ctx.Locals("admin_id")
	idStr, _ := idVal.(string)
	if idStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminSessionMissing, "unauthorized")
	}
	var body request.TwoFASetup
	if err := ctx.Bind().Body(&body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	aid, _ := strconv.ParseInt(idStr, 10, 64)
	result, err := r.auth.StartTwoFactorSetup(ctx.Context(), aid)
	if err != nil {
		r.logger.Error(err, "http - admin - auth - twoFASetup - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorAdminTwoFASetupFailed
		msg := "failed to setup 2fa"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrTwoFASetupNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminTwoFASetupNotFound
			msg = "2fa setup not found"
		case errors.Is(err, authUC.ErrTwoFASetupStore):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorCache
			msg = "cache error"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	dto := response.TwoFASetupStart{SetupID: result.SetupID, Secret: result.Secret, QRCodeImage: result.QRBase64}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(dto))
}

// @Summary 校验并启用 2FA
// @Description 使用 setup_id 从缓存读取密钥验证 OTP，验证通过后写入数据库、生成恢复码并销毁会话。
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.TwoFAVerify true "验证码"
// @Success 200 {object} sharedresp.Envelope{data=response.TwoFAVerifyResult}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/2fa/verify [post]
func (r *Admin) twoFAVerify(ctx fiber.Ctx) error {
	idVal := ctx.Locals("admin_id")
	idStr, _ := idVal.(string)
	if idStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminSessionMissing, "unauthorized")
	}
	var body request.TwoFAVerify
	if err := ctx.Bind().Body(&body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	aid, _ := strconv.ParseInt(idStr, 10, 64)
	result, err := r.auth.VerifyTwoFactorSetup(ctx.Context(), aid, body.SetupID, body.Code)
	if err != nil {
		r.logger.Error(err, "http - admin - auth - twoFAVerify - usecase")
		httpCode := http.StatusInternalServerError
		bizCode := response.ErrorAdminTwoFASetupFailed
		msg := "failed to verify 2fa"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrTwoFASetupNotFound):
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminTwoFASetupNotFound
			msg = "2fa setup not found"
		case errors.Is(err, authUC.ErrOTPWrong):
			httpCode = http.StatusUnauthorized
			bizCode = response.ErrorAdminOTPWrong
			msg = "invalid otp code"
		case errors.Is(err, authUC.ErrTwoFASetupStore):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorCache
			msg = "cache error"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	sess, serr := r.sess.Get(ctx)
	if serr == nil && sess != nil {
		_ = sess.Destroy()
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.TwoFAVerifyResult{
		Enabled:         result.Enabled,
		ReloginRequired: true,
		RecoveryCodes:   result.RecoveryCodes,
	}))
}

// @Summary 禁用 2FA
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.TwoFAAuth true "验证信息"
// @Success 200 {object} sharedresp.Envelope
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/2fa/disable [post]
func (r *Admin) twoFADisable(ctx fiber.Ctx) error {
	idVal := ctx.Locals("admin_id")
	idStr, _ := idVal.(string)
	if idStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminSessionMissing, "unauthorized")
	}
	var body request.TwoFAAuth
	if err := ctx.Bind().Body(&body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	aid, _ := strconv.ParseInt(idStr, 10, 64)
	admin, err := r.auth.GetAdminByID(ctx.Context(), aid)
	if err != nil || admin == nil {
		httpCode := http.StatusUnauthorized
		bizCode := response.ErrorUnauthorized
		msg := "unauthorized"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound) || admin == nil:
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	if admin.TwoFactorSecret == nil || *admin.TwoFactorSecret == "" {
		return sharedresp.WriteSuccess(ctx)
	}
	if body.Code != "" {
		ok, verr := r.auth.ValidateTOTP(ctx.Context(), aid, body.Code)
		if verr != nil {
			r.logger.Error(verr, "http - admin - auth - twoFADisable - otp validate")
			return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "otp validate failed")
		}
		if !ok {
			return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminOTPWrong, "invalid otp code")
		}
	} else if body.RecoveryCode != "" {
		ok, err := r.auth.VerifyAndUseRecoveryCode(ctx.Context(), aid, body.RecoveryCode)
		if err != nil {
			r.logger.Error(err, "http - admin - auth - twoFADisable - recovery verify")
			return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "recovery verify failed")
		}
		if !ok {
			return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminRecoveryCodeWrong, "invalid recovery code")
		}
	} else {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing code or recovery_code")
	}
	if err := r.auth.ClearTwoFactorSecret(ctx.Context(), aid); err != nil {
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorAdminDisable2FAFailed, "failed to disable 2fa")
	}
	return sharedresp.WriteSuccess(ctx)
}

// @Summary 重置 2FA Recovery Codes
// @Tags Admin.Auth
// @Accept json
// @Produce json
// @Security AdminSession
// @Param body body request.TwoFAAuth true "验证信息"
// @Success 200 {object} sharedresp.Envelope{data=response.ResetRecoveryCodes}
// @Failure 400 {object} sharedresp.Envelope
// @Failure 401 {object} sharedresp.Envelope
// @Failure 404 {object} sharedresp.Envelope
// @Failure 500 {object} sharedresp.Envelope
// @Router /admin/auth/2fa/recovery/reset [post]
func (r *Admin) twoFARecoveryReset(ctx fiber.Ctx) error {
	idVal := ctx.Locals("admin_id")
	idStr, _ := idVal.(string)
	if idStr == "" {
		return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminSessionMissing, "unauthorized")
	}
	var body request.TwoFAAuth
	if err := ctx.Bind().Body(&body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	if err := r.validate.Struct(body); err != nil {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamFormat, "invalid request body")
	}
	aid, _ := strconv.ParseInt(idStr, 10, 64)
	admin, err := r.auth.GetAdminByID(ctx.Context(), aid)
	if err != nil || admin == nil {
		httpCode := http.StatusUnauthorized
		bizCode := response.ErrorUnauthorized
		msg := "unauthorized"
		switch {
		case errors.Is(err, authUC.ErrAdminNotFound) || admin == nil:
			httpCode = http.StatusNotFound
			bizCode = response.ErrorAdminNotFound
			msg = "admin not found"
		case errors.Is(err, authUC.ErrRepo):
			httpCode = http.StatusInternalServerError
			bizCode = response.ErrorDatabase
			msg = "database error"
		}
		return sharedresp.WriteError(ctx, httpCode, bizCode, msg)
	}
	if admin.TwoFactorSecret == nil || *admin.TwoFactorSecret == "" {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorAdminTwoFANotEnabled, "2fa not enabled")
	}
	if body.Code != "" {
		ok, verr := r.auth.ValidateTOTP(ctx.Context(), aid, body.Code)
		if verr != nil {
			r.logger.Error(verr, "http - admin - auth - twoFARecoveryReset - otp validate")
			return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "otp validate failed")
		}
		if !ok {
			return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminOTPWrong, "invalid otp code")
		}
	} else if body.RecoveryCode != "" {
		ok, err := r.auth.VerifyAndUseRecoveryCode(ctx.Context(), aid, body.RecoveryCode)
		if err != nil {
			r.logger.Error(err, "http - admin - auth - twoFARecoveryReset - recovery verify")
			return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorSystem, "recovery verify failed")
		}
		if !ok {
			return sharedresp.WriteError(ctx, http.StatusUnauthorized, response.ErrorAdminRecoveryCodeWrong, "invalid recovery code")
		}
	} else {
		return sharedresp.WriteError(ctx, http.StatusBadRequest, response.ErrorParamMissing, "missing code or recovery_code")
	}

	codes, genErr := r.auth.ResetRecoveryCodes(ctx.Context(), aid)
	if genErr != nil {
		r.logger.Error(genErr, "http - admin - auth - twoFARecoveryReset - usecase")
		return sharedresp.WriteError(ctx, http.StatusInternalServerError, response.ErrorAdminRecoveryCodesPersistFailed, "failed to reset recovery codes")
	}
	return sharedresp.WriteSuccess(ctx, sharedresp.WithData(response.ResetRecoveryCodes{RecoveryCodes: codes}))
}
