package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrEmailExists    = errors.New("email exists")
	ErrHashPassword   = errors.New("hash password")
	ErrRepo           = errors.New("repo")
	ErrUserNotFound   = errors.New("user not found")
	ErrPasswordWrong  = errors.New("password wrong")
	ErrTokenSign      = errors.New("token sign")
	ErrTokenInvalid   = errors.New("token invalid")
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenMalformed = errors.New("token malformed")
	ErrTokenMissing   = errors.New("authorization missing")
	ErrUserDisabled   = errors.New("user disabled")
)

type useCase struct {
	repo             repo.UserRepo
	signer           TokenSigner
	refreshStore     repo.RefreshTokenStore
	refreshBlacklist repo.RefreshTokenBlacklistRepo
}

func New(r repo.UserRepo, signer TokenSigner, refreshStore repo.RefreshTokenStore, refreshBlacklist repo.RefreshTokenBlacklistRepo) usecase.UserAuth {
	return &useCase{repo: r, signer: signer, refreshStore: refreshStore, refreshBlacklist: refreshBlacklist}
}

func (uc *useCase) Register(ctx context.Context, username, email, password string) (*output.UserDetail, error) {
	existing, err := uc.repo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, ErrEmailExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHashPassword, err)
	}

	verified := true
	avatar := "/avatar.png"
	bio := "该用户尚未填写个人简介。"
	user := entity.User{
		Name:          username,
		Email:         &email,
		PasswordHash:  string(hashed),
		Avatar:        avatar,
		Bio:           bio,
		EmailVerified: verified,
	}

	id, err := uc.repo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	user.ID = id

	return &output.UserDetail{
		ID:              user.ID,
		Name:            user.Name,
		Email:           user.Email,
		Avatar:          user.Avatar,
		Bio:             user.Bio,
		EmailVerified:   user.EmailVerified,
		Region:          user.Region,
		BlogURL:         user.BlogURL,
		AuthProvider:    user.AuthProvider,
		AuthOpenid:      user.AuthOpenid,
		ShowFullProfile: user.ShowFullProfile,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}, nil
}

func (uc *useCase) Login(ctx context.Context, email, password string) (*output.TokenPair, error) {
	user, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrPasswordWrong
	}
	if user.Status != entity.UserStatusActive {
		return nil, ErrUserDisabled
	}

	accessToken, err := uc.signer.SignAccess(user.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenSign, err)
	}
	refreshToken, err := uc.signer.SignRefresh(user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenSign, err)
	}
	if err := uc.setCurrentRefreshToken(user.ID, refreshToken); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	pair := &output.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        uc.signer.AccessTTL().Milliseconds(),
		RefreshExpiresIn: uc.signer.RefreshTTL().Milliseconds(),
	}
	return pair, nil
}

func (uc *useCase) Refresh(ctx context.Context, refreshToken string) (*output.TokenPair, error) {
	claims, err := uc.signer.ParseRefreshClaims(refreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	uid, err := claims.UserIDInt()
	if err != nil {
		return nil, ErrTokenInvalid
	}
	user, err := uc.repo.GetByID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if user.Status != entity.UserStatusActive {
		return nil, ErrUserDisabled
	}
	if err := uc.validateRefreshToken(ctx, uid, refreshToken); err != nil {
		return nil, err
	}

	accessToken, err := uc.signer.SignAccess(user.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenSign, err)
	}
	newRefreshToken, err := uc.signer.SignRefresh(user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenSign, err)
	}
	if err := uc.setCurrentRefreshToken(user.ID, newRefreshToken); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	exp := time.Now().Add(uc.signer.RefreshTTL())
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	if err := uc.blacklistRefreshToken(ctx, uid, refreshToken, exp); err != nil {
		return nil, err
	}

	pair := &output.TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        uc.signer.AccessTTL().Milliseconds(),
		RefreshExpiresIn: uc.signer.RefreshTTL().Milliseconds(),
	}
	return pair, nil
}

func (uc *useCase) ChangePassword(ctx context.Context, id int64, oldPassword, newPassword string) error {
	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if u == nil {
		return ErrUserNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword)) != nil {
		return ErrPasswordWrong
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHashPassword, err)
	}
	if err := uc.repo.UpdatePasswordHash(ctx, id, string(hashed)); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func (uc *useCase) RevokeUserRefreshToken(ctx context.Context, userID int64) error {
	if uc.refreshStore == nil && uc.refreshBlacklist == nil {
		return nil
	}
	current := ""
	if uc.refreshStore != nil {
		current = uc.refreshStore.Get(userID)
	}
	if current == "" {
		return nil
	}
	exp := time.Now().Add(uc.signer.RefreshTTL())
	if uc.refreshBlacklist != nil {
		if err := uc.refreshBlacklist.Add(ctx, userID, hashToken(current), exp); err != nil {
			return fmt.Errorf("%w: %v", ErrRepo, err)
		}
	}
	if uc.refreshStore != nil {
		if err := uc.refreshStore.Delete(userID); err != nil {
			return fmt.Errorf("%w: %v", ErrRepo, err)
		}
	}
	return nil
}

func (uc *useCase) ValidateSession(ctx context.Context, userID int64, refreshToken string) error {
	claims, err := uc.signer.ParseRefreshClaims(refreshToken)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) {
			return ErrTokenExpired
		}
		return ErrTokenInvalid
	}
	uid, err := claims.UserIDInt()
	if err != nil || uid != userID {
		return ErrTokenInvalid
	}

	if uc.refreshStore != nil {
		current := uc.refreshStore.Get(userID)
		if current != "" {
			if current != refreshToken {
				return ErrTokenInvalid
			}
			return nil
		}
	}

	u, err := uc.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if u == nil {
		return ErrUserNotFound
	}
	if u.Status != entity.UserStatusActive {
		return ErrUserDisabled
	}

	if err := uc.validateRefreshToken(ctx, userID, refreshToken); err != nil {
		return err
	}
	if uc.refreshStore != nil {
		exp := time.Now().Add(uc.signer.RefreshTTL())
		if claims.ExpiresAt != nil {
			exp = claims.ExpiresAt.Time
		}
		ttl := time.Until(exp)
		if ttl > 0 {
			if err := uc.refreshStore.Set(userID, refreshToken, ttl); err != nil {
				return fmt.Errorf("%w: %v", ErrRepo, err)
			}
		}
	}
	return nil
}

func (uc *useCase) ResetPasswordByEmail(ctx context.Context, email, newPassword string) error {
	u, err := uc.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if u == nil {
		return ErrUserNotFound
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrHashPassword, err)
	}
	if err := uc.repo.UpdatePasswordHash(ctx, u.ID, string(hashed)); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func (uc *useCase) setCurrentRefreshToken(userID int64, token string) error {
	if uc.refreshStore == nil {
		return nil
	}
	return uc.refreshStore.Set(userID, token, uc.signer.RefreshTTL())
}

func (uc *useCase) validateRefreshToken(ctx context.Context, userID int64, token string) error {
	if uc.refreshStore != nil {
		current := uc.refreshStore.Get(userID)
		if current != "" {
			if current != token {
				return ErrTokenInvalid
			}
			return nil
		}
	}
	if uc.refreshBlacklist == nil {
		return nil
	}
	blacklisted, err := uc.refreshBlacklist.Exists(ctx, hashToken(token))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	if blacklisted {
		return ErrTokenInvalid
	}
	return nil
}

func (uc *useCase) blacklistRefreshToken(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	if uc.refreshBlacklist == nil {
		return nil
	}
	if err := uc.refreshBlacklist.Add(ctx, userID, hashToken(token), expiresAt); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
