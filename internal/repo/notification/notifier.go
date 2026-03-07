package notification

import (
	"context"
	"encoding/json"
	"unicode/utf8"

	"github.com/scc749/nimbus-blog-api/internal/entity"
	"github.com/scc749/nimbus-blog-api/internal/repo"
	"github.com/scc749/nimbus-blog-api/pkg/ssehub"
)

const _maxNotificationContentLen = 100

type notifier struct {
	notifications repo.NotificationRepo
	hub           *ssehub.Hub
}

func NewNotifier(notifications repo.NotificationRepo, hub *ssehub.Hub) repo.Notifier {
	return &notifier{notifications: notifications, hub: hub}
}

func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen]) + "..."
}

func (n *notifier) Send(ctx context.Context, notif entity.Notification) error {
	if utf8.RuneCountInString(notif.Content) > _maxNotificationContentLen {
		notif.Content = truncate(notif.Content, _maxNotificationContentLen)
	}
	id, err := n.notifications.Create(ctx, notif)
	if err != nil {
		return err
	}
	notif.ID = id

	data, _ := json.Marshal(notif)
	n.hub.Publish(notif.UserID, ssehub.Event{Name: "notification", Data: data})

	count, _ := n.notifications.CountUnread(ctx, notif.UserID)
	countData, _ := json.Marshal(map[string]int64{"count": count})
	n.hub.Publish(notif.UserID, ssehub.Event{Name: "unread_count", Data: countData})

	return nil
}
