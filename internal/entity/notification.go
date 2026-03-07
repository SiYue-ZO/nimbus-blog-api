package entity

import (
	"encoding/json"
	"time"
)

const (
	NotificationTypeCommentReply    = "comment_reply"
	NotificationTypeCommentApproved = "comment_approved"
	NotificationTypeAdminMessage    = "admin_message"
)

type Notification struct {
	ID        int64
	UserID    int64
	Type      string
	Title     string
	Content   string
	Meta      json.RawMessage
	IsRead    bool
	CreatedAt time.Time
}
