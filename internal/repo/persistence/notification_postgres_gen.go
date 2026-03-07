package persistence

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/model"
	"github.com/scc749/nimbus-blog-api/internal/repo/persistence/gen/query"
	"gorm.io/gorm"
)

type notificationRepo struct {
	query *query.Query
}

func NewNotificationRepo(db *gorm.DB) repo.NotificationRepo {
	return &notificationRepo{query: query.Use(db)}
}

func (r *notificationRepo) Create(ctx context.Context, en entity.Notification) (int64, error) {
	mn := toModelNotification(&en)
	if err := r.query.Notification.WithContext(ctx).Create(mn); err != nil {
		return 0, err
	}
	return mn.ID, nil
}

func (r *notificationRepo) List(ctx context.Context, offset, limit int, userID int64, isRead *bool, sortBy *string, order *string) ([]*entity.Notification, int64, error) {
	n := r.query.Notification
	do := n.WithContext(ctx).Where(n.UserID.Eq(userID))

	if isRead != nil {
		do = do.Where(n.IsRead.Is(*isRead))
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	if sortBy != nil && *sortBy != "" {
		orderField, ok := n.GetFieldByName(*sortBy)
		if ok {
			if order != nil && strings.EqualFold(*order, "asc") {
				do = do.Order(orderField)
			} else {
				do = do.Order(orderField.Desc())
			}
		}
	} else {
		do = do.Order(n.CreatedAt.Desc())
	}

	rows, err := do.Offset(offset).Limit(limit).Find()
	if err != nil {
		return nil, 0, err
	}

	items := make([]*entity.Notification, len(rows))
	for i, mn := range rows {
		items[i] = toEntityNotification(mn)
	}
	return items, total, nil
}

func (r *notificationRepo) CountUnread(ctx context.Context, userID int64) (int64, error) {
	n := r.query.Notification
	return n.WithContext(ctx).Where(n.UserID.Eq(userID), n.IsRead.Is(false)).Count()
}

func (r *notificationRepo) MarkRead(ctx context.Context, id, userID int64) error {
	n := r.query.Notification
	_, err := n.WithContext(ctx).Where(n.ID.Eq(id), n.UserID.Eq(userID)).Update(n.IsRead, true)
	return err
}

func (r *notificationRepo) MarkAllRead(ctx context.Context, userID int64) error {
	n := r.query.Notification
	_, err := n.WithContext(ctx).Where(n.UserID.Eq(userID), n.IsRead.Is(false)).Update(n.IsRead, true)
	return err
}

func (r *notificationRepo) Delete(ctx context.Context, id, userID int64) error {
	n := r.query.Notification
	_, err := n.WithContext(ctx).Where(n.ID.Eq(id), n.UserID.Eq(userID)).Delete()
	return err
}

func toModelNotification(en *entity.Notification) *model.Notification {
	meta := en.Meta
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return &model.Notification{
		ID:        en.ID,
		UserID:    en.UserID,
		Type:      en.Type,
		Title:     en.Title,
		Content:   en.Content,
		Meta:      string(meta),
		IsRead:    en.IsRead,
		CreatedAt: en.CreatedAt,
	}
}

func toEntityNotification(mn *model.Notification) *entity.Notification {
	return &entity.Notification{
		ID:        mn.ID,
		UserID:    mn.UserID,
		Type:      mn.Type,
		Title:     mn.Title,
		Content:   mn.Content,
		Meta:      json.RawMessage(mn.Meta),
		IsRead:    mn.IsRead,
		CreatedAt: mn.CreatedAt,
	}
}
