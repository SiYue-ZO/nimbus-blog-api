package persistence

import (
	"context"
	"strings"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type userRepo struct {
	query *query.Query
}

func NewUserRepo(db *gorm.DB) repo.UserRepo {
	return &userRepo{query: query.Use(db)}
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	u := r.query.User
	mu, err := u.WithContext(ctx).Where(u.Email.Eq(email)).First()
	if err != nil {
		return nil, err
	}
	return toEntityUser(mu), nil
}

func (r *userRepo) Create(ctx context.Context, eu entity.User) (int64, error) {
	mu := toModelUser(&eu)
	if err := r.query.User.WithContext(ctx).Create(mu); err != nil {
		return 0, err
	}
	return mu.ID, nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	u := r.query.User
	mu, err := u.WithContext(ctx).Where(u.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return toEntityUser(mu), nil
}

func (r *userRepo) GetByIDs(ctx context.Context, ids []int64) ([]*entity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	u := r.query.User
	rows, err := u.WithContext(ctx).Where(u.ID.In(ids...)).Find()
	if err != nil {
		return nil, err
	}
	users := make([]*entity.User, len(rows))
	for i, mu := range rows {
		users[i] = toEntityUser(mu)
	}
	return users, nil
}

func (r *userRepo) Update(ctx context.Context, eu entity.User) error {
	u := r.query.User
	mu := toModelUser(&eu)
	_, err := u.WithContext(ctx).Where(u.ID.Eq(eu.ID)).Updates(mu)
	return err
}

func (r *userRepo) UpdatePasswordHash(ctx context.Context, id int64, newHash string) error {
	u := r.query.User
	_, err := u.WithContext(ctx).Where(u.ID.Eq(id)).Update(u.PasswordHash, newHash)
	return err
}

func (r *userRepo) List(ctx context.Context, offset, limit int, status *string, keyword *string, sortBy *string, order *string) ([]*entity.User, int64, error) {
	u := r.query.User
	do := u.WithContext(ctx)

	if status != nil && *status != "" {
		do = do.Where(u.Status.Eq(*status))
	}
	if keyword != nil && *keyword != "" {
		do = do.Where(field.NewUnsafeFieldRaw("name ILIKE ? OR email ILIKE ?", "%"+*keyword+"%", "%"+*keyword+"%"))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if sortBy != nil && *sortBy != "" {
		orderField, ok := u.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else {
		do = do.Order(u.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	users := make([]*entity.User, len(rows))
	for i, mu := range rows {
		users[i] = toEntityUser(mu)
	}
	return users, total, nil
}

func (r *userRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	u := r.query.User
	_, err := u.WithContext(ctx).Where(u.ID.Eq(id)).Update(u.Status, status)
	return err
}

func toModelUser(eu *entity.User) *model.User {
	return &model.User{
		ID:              eu.ID,
		Name:            eu.Name,
		Email:           eu.Email,
		PasswordHash:    eu.PasswordHash,
		Avatar:          eu.Avatar,
		Bio:             eu.Bio,
		Status:          eu.Status,
		EmailVerified:   eu.EmailVerified,
		Region:          eu.Region,
		BlogURL:         eu.BlogURL,
		AuthProvider:    eu.AuthProvider,
		AuthOpenid:      eu.AuthOpenid,
		ShowFullProfile: eu.ShowFullProfile,
		CreatedAt:       eu.CreatedAt,
		UpdatedAt:       eu.UpdatedAt,
	}
}

func toEntityUser(mu *model.User) *entity.User {
	return &entity.User{
		ID:              mu.ID,
		Name:            mu.Name,
		Email:           mu.Email,
		PasswordHash:    mu.PasswordHash,
		Avatar:          mu.Avatar,
		Bio:             mu.Bio,
		Status:          mu.Status,
		EmailVerified:   mu.EmailVerified,
		Region:          mu.Region,
		BlogURL:         mu.BlogURL,
		AuthProvider:    mu.AuthProvider,
		AuthOpenid:      mu.AuthOpenid,
		ShowFullProfile: mu.ShowFullProfile,
		CreatedAt:       mu.CreatedAt,
		UpdatedAt:       mu.UpdatedAt,
	}
}
