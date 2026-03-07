package repo

import (
	"context"
	"time"

	"github.com/scc749/nimbus-blog-api/internal/entity"
)

type (
	CaptchaStore interface {
		Set(id string, value string) error
		Get(id string, clear bool) string
		Verify(id, answer string, clear bool) bool
	}

	EmailCodeStore interface {
		Set(id string, value string) error
		Get(id string, clear bool) string
		Verify(id, value string, clear bool) bool
	}

	EmailSender interface {
		Send(to string, subject string, body string) error
	}

	ObjectStore interface {
		PresignUpload(ctx context.Context, bucket, key string, expires time.Duration, contentType string) (string, error)
		PresignDownload(ctx context.Context, bucket, key string, expires time.Duration) (string, error)
		Delete(ctx context.Context, bucket, key string) error
	}

	AdminRepo interface {
		GetByUsername(ctx context.Context, username string) (*entity.Admin, error)
		GetByID(ctx context.Context, id int64) (*entity.Admin, error)
		UpdatePasswordHash(ctx context.Context, id int64, newHash string, clearResetFlag bool) error
		SetTwoFactorSecret(ctx context.Context, id int64, secret string) error
		ClearTwoFactorSecret(ctx context.Context, id int64) error
		CreateRecoveryCodes(ctx context.Context, id int64, hashes []string) error
		VerifyAndUseRecoveryCode(ctx context.Context, id int64, code string) (bool, error)
		InvalidateRecoveryCodes(ctx context.Context, id int64) error
		UpdateProfile(ctx context.Context, id int64, nickname, specialization string) error
	}

	UserRepo interface {
		GetByEmail(ctx context.Context, email string) (*entity.User, error)
		Create(ctx context.Context, u entity.User) (int64, error)
		GetByID(ctx context.Context, id int64) (*entity.User, error)
		GetByIDs(ctx context.Context, ids []int64) ([]*entity.User, error)
		Update(ctx context.Context, u entity.User) error
		UpdatePasswordHash(ctx context.Context, id int64, newHash string) error
		List(ctx context.Context, offset, limit int, status *string, keyword *string, sortBy *string, order *string) ([]*entity.User, int64, error)
		UpdateStatus(ctx context.Context, id int64, status string) error
	}

	RefreshTokenStore interface {
		Set(userID int64, token string, ttl time.Duration) error
		Get(userID int64) string
		Delete(userID int64) error
	}

	AdminTwoFASetupStore interface {
		Set(setupID string, adminID int64, secret string, ttl time.Duration) error
		Get(setupID string) (adminID int64, secret string, ok bool)
		Delete(setupID string) error
	}

	RefreshTokenBlacklistRepo interface {
		Add(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
		Exists(ctx context.Context, tokenHash string) (bool, error)
	}

	TranslationWebAPI interface {
		Translate(ctx context.Context, text, source, destination string) (string, error)
	}

	LLMWebAPI interface {
		Complete(ctx context.Context, system string, user string) (string, error)
	}

	PostRepo interface {
		List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string, categoryID *int, tagID *int, status *string, isFeatured *bool, featuredFirst bool) ([]*entity.Post, int64, error)
		GetByID(ctx context.Context, id int64) (*entity.Post, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Post, error)
		Create(ctx context.Context, p entity.Post) (int64, error)
		Update(ctx context.Context, p entity.Post) error
		Delete(ctx context.Context, id int64) error
		SetTags(ctx context.Context, postID int64, tagIDs []int64) error
	}

	TagRepo interface {
		List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string) ([]*entity.Tag, int64, error)
		ListAll(ctx context.Context) ([]*entity.Tag, error)
		GetByID(ctx context.Context, id int64) (*entity.Tag, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Tag, error)
		ListByPostID(ctx context.Context, postID int64) ([]*entity.Tag, error)
		Create(ctx context.Context, t entity.Tag) (int64, error)
		Update(ctx context.Context, t entity.Tag) error
		Delete(ctx context.Context, id int64) error
	}

	CategoryRepo interface {
		List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string) ([]*entity.Category, int64, error)
		ListAll(ctx context.Context) ([]*entity.Category, error)
		GetByID(ctx context.Context, id int64) (*entity.Category, error)
		GetBySlug(ctx context.Context, slug string) (*entity.Category, error)
		Create(ctx context.Context, c entity.Category) (int64, error)
		Update(ctx context.Context, c entity.Category) error
		Delete(ctx context.Context, id int64) error
	}

	CommentRepo interface {
		List(ctx context.Context, offset, limit int, status *string, sortBy *string, order *string) ([]*entity.Comment, int64, error)
		GetByID(ctx context.Context, id int64) (*entity.Comment, error)
		Create(ctx context.Context, c entity.Comment) (int64, error)
		UpdateStatus(ctx context.Context, id int64, status string) error
		Delete(ctx context.Context, id int64) error
		ListApprovedByPostID(ctx context.Context, postID int64) ([]*entity.Comment, error)
	}

	PostLikeRepo interface {
		Toggle(ctx context.Context, postID, userID int64) (liked bool, count int32, err error)
		Remove(ctx context.Context, postID, userID int64) (removed bool, count int32, err error)
		HasLiked(ctx context.Context, postID, userID int64) (bool, error)
	}

	PostViewRepo interface {
		Record(ctx context.Context, pv entity.PostView) error
	}

	CommentLikeRepo interface {
		Toggle(ctx context.Context, commentID, userID int64) (liked bool, count int32, err error)
		Remove(ctx context.Context, commentID, userID int64) (removed bool, count int32, err error)
		HasLiked(ctx context.Context, commentID, userID int64) (bool, error)
	}

	FeedbackRepo interface {
		List(ctx context.Context, offset, limit int, status *string, sortBy *string, order *string) ([]*entity.Feedback, int64, error)
		GetByID(ctx context.Context, id int64) (*entity.Feedback, error)
		Create(ctx context.Context, f entity.Feedback) (int64, error)
		UpdateStatus(ctx context.Context, id int64, status string) error
		Delete(ctx context.Context, id int64) error
	}

	LinkRepo interface {
		List(ctx context.Context, offset, limit int, keyword *string, sortBy *string, order *string) ([]*entity.Link, int64, error)
		Create(ctx context.Context, l entity.Link) (int64, error)
		Update(ctx context.Context, l entity.Link) error
		Delete(ctx context.Context, id int64) error
		ListAllPublic(ctx context.Context) ([]*entity.Link, error)
	}

	SiteSettingRepo interface {
		ListAll(ctx context.Context) ([]*entity.SiteSetting, error)
		GetByKey(ctx context.Context, key string) (*entity.SiteSetting, error)
		Upsert(ctx context.Context, s entity.SiteSetting) error
	}

	FileRepo interface {
		Create(ctx context.Context, f entity.File) (int64, error)
		GetByObjectKey(ctx context.Context, objectKey string) (*entity.File, error)
		List(ctx context.Context, offset, limit int, usage *string, sortBy *string, order *string) ([]*entity.File, int64, error)
		ListByResource(ctx context.Context, usage string, resourceID int64) ([]*entity.File, error)
		UpdateResourceID(ctx context.Context, objectKey string, resourceID int64) error
		ClearResourceIDByResourceAndUsage(ctx context.Context, resourceID int64, usage string) error
		DeleteByObjectKey(ctx context.Context, objectKey string) error
	}

	NotificationRepo interface {
		Create(ctx context.Context, n entity.Notification) (int64, error)
		List(ctx context.Context, offset, limit int, userID int64, isRead *bool, sortBy *string, order *string) ([]*entity.Notification, int64, error)
		MarkRead(ctx context.Context, id, userID int64) error
		MarkAllRead(ctx context.Context, userID int64) error
		CountUnread(ctx context.Context, userID int64) (int64, error)
		Delete(ctx context.Context, id, userID int64) error
	}

	Notifier interface {
		Send(ctx context.Context, n entity.Notification) error
	}
)
