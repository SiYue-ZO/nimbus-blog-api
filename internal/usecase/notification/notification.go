package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/internal/usecase"
	"github.com/scc749/nimbus-blog-api/internal/usecase/input"
	"github.com/scc749/nimbus-blog-api/internal/usecase/output"
	"github.com/scc749/nimbus-blog-api/pkg/ssehub"
)

var ErrRepo = errors.New("repo")

type useCase struct {
	notifications repo.NotificationRepo
	notifier      repo.Notifier
	hub           *ssehub.Hub
}

func New(notifications repo.NotificationRepo, notifier repo.Notifier, hub *ssehub.Hub) usecase.Notification {
	return &useCase{notifications: notifications, notifier: notifier, hub: hub}
}

func (u *useCase) Subscribe(userID int64) chan ssehub.Event {
	return u.hub.Subscribe(userID)
}

func (u *useCase) Unsubscribe(userID int64, ch chan ssehub.Event) {
	u.hub.Unsubscribe(userID, ch)
}

func (u *useCase) GetUnreadCount(ctx context.Context, userID int64) (int64, error) {
	count, err := u.notifications.CountUnread(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrRepo, err)
	}
	return count, nil
}

func (u *useCase) ListMyNotifications(ctx context.Context, params input.ListNotifications) (*output.ListResult[output.NotificationDetail], error) {
	offset := (params.Page - 1) * params.PageSize

	var isRead *bool
	if params.IsRead != nil {
		isRead = (*bool)(params.IsRead)
	}
	var sortBy, order *string
	if params.Sort != nil {
		sortBy = &params.Sort.SortBy
		order = &params.Sort.Order
	}

	rows, total, err := u.notifications.List(ctx, offset, params.PageSize, params.UserID, isRead, sortBy, order)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepo, err)
	}

	items := make([]output.NotificationDetail, len(rows))
	for i, n := range rows {
		items[i] = toNotificationDetail(n)
	}

	return &output.ListResult[output.NotificationDetail]{
		Items:    items,
		Page:     params.Page,
		PageSize: params.PageSize,
		Total:    total,
	}, nil
}

func (u *useCase) MarkRead(ctx context.Context, id, userID int64) error {
	if err := u.notifications.MarkRead(ctx, id, userID); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	u.publishUnreadCount(ctx, userID)
	return nil
}

func (u *useCase) MarkAllRead(ctx context.Context, userID int64) error {
	if err := u.notifications.MarkAllRead(ctx, userID); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	u.publishUnreadCount(ctx, userID)
	return nil
}

func (u *useCase) DeleteNotification(ctx context.Context, id, userID int64) error {
	if err := u.notifications.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("%w: %v", ErrRepo, err)
	}
	u.publishUnreadCount(ctx, userID)
	return nil
}

func (u *useCase) SendAdminMessage(ctx context.Context, params input.SendAdminNotification) error {
	meta, _ := json.Marshal(map[string]string{entity.NotificationMetaSource: "admin"})
	return u.notifier.Send(ctx, entity.Notification{
		UserID:  params.UserID,
		Type:    entity.NotificationTypeAdminMessage,
		Title:   params.Title,
		Content: params.Content,
		Meta:    meta,
	})
}

func toNotificationDetail(n *entity.Notification) output.NotificationDetail {
	if n == nil {
		return output.NotificationDetail{}
	}

	meta := n.Meta
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}

	d := output.NotificationDetail{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Content:   n.Content,
		Meta:      meta,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}

	if len(n.Meta) > 0 {
		var m struct {
			PostSlug  *string `json:"post_slug"`
			CommentID *int64  `json:"comment_id"`
			TargetURL *string `json:"target_url"`
		}
		if err := json.Unmarshal(n.Meta, &m); err == nil {
			d.PostSlug = m.PostSlug
			d.CommentID = m.CommentID
			d.TargetURL = m.TargetURL
		}
	}

	return d
}

func (u *useCase) publishUnreadCount(ctx context.Context, userID int64) {
	count, err := u.notifications.CountUnread(ctx, userID)
	if err != nil {
		return
	}
	data, err := json.Marshal(map[string]int64{"count": count})
	if err != nil {
		return
	}
	u.hub.Publish(userID, ssehub.Event{Name: "unread_count", Data: data})
}
