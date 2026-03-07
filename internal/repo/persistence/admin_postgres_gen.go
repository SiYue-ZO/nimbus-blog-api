package persistence

import (
	"context"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type adminRepo struct {
	query *query.Query
}

func NewAdminRepo(db *gorm.DB) repo.AdminRepo {
	query.SetDefault(db)
	return &adminRepo{query: query.Q}
}

func (r *adminRepo) GetByUsername(ctx context.Context, username string) (*entity.Admin, error) {
	ma, err := r.query.Admin.WithContext(ctx).Where(query.Admin.Username.Eq(username)).First()
	if err != nil {
		return nil, err
	}
	return toEntityAdmin(ma), nil
}

func (r *adminRepo) GetByID(ctx context.Context, id int64) (*entity.Admin, error) {
	ma, err := r.query.Admin.WithContext(ctx).Where(query.Admin.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return toEntityAdmin(ma), nil
}

func (r *adminRepo) UpdatePasswordHash(ctx context.Context, id int64, newHash string, clearResetFlag bool) error {
	vals := map[string]interface{}{"password_hash": newHash}
	if clearResetFlag {
		vals["must_reset_password"] = false
	}
	if _, err := r.query.Admin.WithContext(ctx).Where(query.Admin.ID.Eq(id)).Updates(vals); err != nil {
		return err
	}
	return nil
}

func (r *adminRepo) SetTwoFactorSecret(ctx context.Context, id int64, secret string) error {
	_, err := r.query.Admin.WithContext(ctx).Where(query.Admin.ID.Eq(id)).Update(query.Admin.TwoFactorSecret, &secret)
	return err
}
func (r *adminRepo) ClearTwoFactorSecret(ctx context.Context, id int64) error {
	_, err := r.query.Admin.WithContext(ctx).Where(query.Admin.ID.Eq(id)).Update(query.Admin.TwoFactorSecret, nil)
	return err
}

func (r *adminRepo) CreateRecoveryCodes(ctx context.Context, id int64, hashes []string) error {
	if len(hashes) == 0 {
		return nil
	}
	codes := make([]*model.AdminRecoveryCode, 0, len(hashes))
	for _, h := range hashes {
		codes = append(codes, &model.AdminRecoveryCode{AdminID: id, CodeHash: h})
	}
	return r.query.AdminRecoveryCode.WithContext(ctx).CreateInBatches(codes, 100)
}

func (r *adminRepo) VerifyAndUseRecoveryCode(ctx context.Context, id int64, code string) (bool, error) {
	codes, err := r.query.AdminRecoveryCode.WithContext(ctx).Where(query.AdminRecoveryCode.AdminID.Eq(id)).Where(query.AdminRecoveryCode.UsedAt.IsNull()).Find()
	if err != nil {
		return false, err
	}
	for _, c := range codes {
		if bcrypt.CompareHashAndPassword([]byte(c.CodeHash), []byte(code)) == nil {
			now := time.Now()
			_, err := r.query.AdminRecoveryCode.WithContext(ctx).Where(query.AdminRecoveryCode.ID.Eq(c.ID)).Update(query.AdminRecoveryCode.UsedAt, &now)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (r *adminRepo) UpdateProfile(ctx context.Context, id int64, nickname, specialization string) error {
	_, err := r.query.Admin.WithContext(ctx).Where(query.Admin.ID.Eq(id)).Updates(map[string]interface{}{
		"nickname":       nickname,
		"specialization": specialization,
	})
	return err
}

func (r *adminRepo) InvalidateRecoveryCodes(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.query.AdminRecoveryCode.WithContext(ctx).Where(query.AdminRecoveryCode.AdminID.Eq(id)).Where(query.AdminRecoveryCode.UsedAt.IsNull()).Update(query.AdminRecoveryCode.UsedAt, &now)
	return err
}

func toEntityAdmin(ma *model.Admin) *entity.Admin {
	return &entity.Admin{
		ID:                ma.ID,
		Username:          ma.Username,
		PasswordHash:      ma.PasswordHash,
		Nickname:          ma.Nickname,
		Specialization:    ma.Specialization,
		MustResetPassword: ma.MustResetPassword,
		TwoFactorSecret:   ma.TwoFactorSecret,
		CreatedAt:         ma.CreatedAt,
		UpdatedAt:         ma.UpdatedAt,
	}
}
