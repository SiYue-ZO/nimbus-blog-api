package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type useCase struct {
	repo       repo.AdminRepo
	setupStore repo.AdminTwoFASetupStore
	issuer     string
	totp       TOTP
	enc        Encryptor
}

var (
	ErrAdminNotFound      = errors.New("admin not found")
	ErrRepo               = errors.New("repo")
	ErrPasswordWrong      = errors.New("password wrong")
	ErrHashPassword       = errors.New("hash password")
	ErrEncrypt            = errors.New("encrypt")
	ErrTwoFANotEnabled    = errors.New("2fa not enabled")
	ErrTwoFASetup         = errors.New("2fa setup")
	ErrTwoFASetupStore    = errors.New("2fa setup store")
	ErrTwoFASetupNotFound = errors.New("2fa setup not found")
	ErrOTPWrong           = errors.New("otp wrong")
)

func New(r repo.AdminRepo, setupStore repo.AdminTwoFASetupStore, issuer string, totpProv TOTP, enc Encryptor) usecase.AdminAuth {
	return &useCase{repo: r, setupStore: setupStore, issuer: issuer, totp: totpProv, enc: enc}
}

func (uc *useCase) Login(ctx context.Context, username, password string) (*output.AdminDetail, error) {
	a, err := uc.repo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if a == nil {
		return nil, ErrAdminNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)) != nil {
		return nil, ErrPasswordWrong
	}
	return toAdminDetail(a), nil
}

func (uc *useCase) ChangePassword(ctx context.Context, params input.ChangePassword) error {
	a, err := uc.repo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminNotFound
		}
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(params.OldPassword)) != nil {
		return ErrPasswordWrong
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(params.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHashPassword, err)
	}
	return uc.repo.UpdatePasswordHash(ctx, params.ID, string(hashed), params.ClearResetFlag)
}

func (uc *useCase) SetTwoFactorSecret(ctx context.Context, id int64, secret string) error {
	encSecret, err := uc.enc.Encrypt([]byte(secret))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEncrypt, err)
	}
	return uc.repo.SetTwoFactorSecret(ctx, id, encSecret)
}

func (uc *useCase) ClearTwoFactorSecret(ctx context.Context, id int64) error {
	return uc.repo.ClearTwoFactorSecret(ctx, id)
}

func (uc *useCase) GetAdminByID(ctx context.Context, id int64) (*output.AdminDetail, error) {
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return toAdminDetail(a), nil
}

func (uc *useCase) GetProfile(ctx context.Context, id int64) (*output.AdminProfile, error) {
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return &output.AdminProfile{
		Nickname:       a.Nickname,
		Specialization: a.Specialization,
		TwoFAEnabled:   a.TwoFactorSecret != nil && *a.TwoFactorSecret != "",
	}, nil
}

func (uc *useCase) UpdateProfile(ctx context.Context, params input.UpdateAdminProfile) error {
	_, err := uc.repo.GetByID(ctx, params.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrAdminNotFound
		}
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return uc.repo.UpdateProfile(ctx, params.ID, params.Nickname, params.Specialization)
}

func (uc *useCase) VerifyAndUseRecoveryCode(ctx context.Context, id int64, code string) (bool, error) {
	ok, err := uc.repo.VerifyAndUseRecoveryCode(ctx, id, code)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return ok, nil
}

func (uc *useCase) InvalidateRecoveryCodes(ctx context.Context, id int64) error {
	return uc.repo.InvalidateRecoveryCodes(ctx, id)
}

func (uc *useCase) ValidateTOTP(ctx context.Context, id int64, code string) (bool, error) {
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, ErrAdminNotFound
		}
		return false, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if a.TwoFactorSecret == nil || *a.TwoFactorSecret == "" {
		return false, ErrTwoFANotEnabled
	}
	bs, derr := uc.enc.Decrypt(*a.TwoFactorSecret)
	if derr != nil {
		return false, fmt.Errorf("%w: %v", ErrEncrypt, derr)
	}
	ok := uc.totp.Validate(code, string(bs))
	return ok, nil
}

func (uc *useCase) StartTwoFactorSetup(ctx context.Context, id int64) (*output.TwoFASetupStart, error) {
	if uc.setupStore == nil {
		return nil, ErrTwoFASetupStore
	}
	a, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAdminNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	secret, b64, genErr := uc.totp.Generate(uc.issuer, a.Username)
	if genErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrTwoFASetup, genErr)
	}

	setupID, sidErr := generateSetupID(16)
	if sidErr != nil {
		return nil, fmt.Errorf("%w: %v", ErrTwoFASetup, sidErr)
	}
	if err := uc.setupStore.Set(setupID, id, secret, 10*time.Minute); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTwoFASetupStore, err)
	}

	return &output.TwoFASetupStart{SetupID: setupID, Secret: secret, QRBase64: b64}, nil
}

func (uc *useCase) VerifyTwoFactorSetup(ctx context.Context, id int64, setupID string, code string) (*output.TwoFAVerifyResult, error) {
	if uc.setupStore == nil {
		return nil, ErrTwoFASetupStore
	}
	sid, secret, ok := uc.setupStore.Get(setupID)
	if !ok || sid != id {
		return nil, ErrTwoFASetupNotFound
	}
	if !uc.totp.Validate(code, secret) {
		return nil, ErrOTPWrong
	}

	encSecret, eerr := uc.enc.Encrypt([]byte(secret))
	if eerr != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncrypt, eerr)
	}
	if err := uc.repo.SetTwoFactorSecret(ctx, id, encSecret); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if err := uc.repo.InvalidateRecoveryCodes(ctx, id); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	codes := make([]string, 0, 8)
	hashes := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		c, genErr := generateRecoveryCode(10)
		if genErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrTwoFASetup, genErr)
		}
		codes = append(codes, c)
		h, hashErr := bcrypt.GenerateFromPassword([]byte(c), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrTwoFASetup, hashErr)
		}
		hashes = append(hashes, string(h))
	}
	if err := uc.repo.CreateRecoveryCodes(ctx, id, hashes); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	_ = uc.setupStore.Delete(setupID)

	return &output.TwoFAVerifyResult{Enabled: true, RecoveryCodes: codes}, nil
}

func (uc *useCase) ResetRecoveryCodes(ctx context.Context, id int64) ([]string, error) {
	if err := uc.repo.InvalidateRecoveryCodes(ctx, id); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	codes := make([]string, 0, 8)
	hashes := make([]string, 0, 8)
	for i := 0; i < 8; i++ {
		c, genErr := generateRecoveryCode(10)
		if genErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrTwoFASetup, genErr)
		}
		codes = append(codes, c)
		h, hashErr := bcrypt.GenerateFromPassword([]byte(c), bcrypt.DefaultCost)
		if hashErr != nil {
			return nil, fmt.Errorf("%w: %v", ErrTwoFASetup, hashErr)
		}
		hashes = append(hashes, string(h))
	}
	if err := uc.repo.CreateRecoveryCodes(ctx, id, hashes); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return codes, nil
}

func toAdminDetail(a *entity.Admin) *output.AdminDetail {
	return &output.AdminDetail{
		ID:                a.ID,
		Username:          a.Username,
		PasswordHash:      a.PasswordHash,
		Nickname:          a.Nickname,
		Specialization:    a.Specialization,
		MustResetPassword: a.MustResetPassword,
		TwoFactorSecret:   a.TwoFactorSecret,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
	}
}

func generateRecoveryCode(n int) (string, error) {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := 0; i < n; i++ {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b), nil
}

func generateSetupID(nbytes int) (string, error) {
	if nbytes <= 0 {
		nbytes = 16
	}
	b := make([]byte, nbytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
