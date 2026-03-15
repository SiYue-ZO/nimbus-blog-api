package output

import (
	"encoding/json"
	"time"
)

type NotificationDetail struct {
	ID        int64
	Type      string
	Title     string
	Content   string
	Meta      json.RawMessage
	PostSlug  *string
	CommentID *int64
	TargetURL *string
	IsRead    bool
	CreatedAt time.Time
}
