package response

import (
	"encoding/json"
	"time"
)

type NotificationDetail struct {
	ID        int64           `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Content   string          `json:"content"`
	Meta      json.RawMessage `json:"meta"`
	PostSlug  *string         `json:"post_slug,omitempty"`
	CommentID *int64          `json:"comment_id,omitempty"`
	TargetURL *string         `json:"target_url,omitempty"`
	IsRead    bool            `json:"is_read"`
	CreatedAt time.Time       `json:"created_at"`
}

type UnreadCount struct {
	Count int64 `json:"count"`
}
