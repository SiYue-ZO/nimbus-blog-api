package response

import "time"

type CommentDetail struct {
	ID           int64     `json:"id"`
	PostID       int64     `json:"post_id"`
	ParentID     *int64    `json:"parent_id"`
	UserID       int64     `json:"user_id"`
	Content      string    `json:"content"`
	Likes        int32     `json:"likes"`
	RepliesCount int32     `json:"replies_count"`
	Status       string    `json:"status"`
	IPAddress    *string   `json:"ip_address"`
	UserAgent    *string   `json:"user_agent"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	UserProfile  `json:"user_profile"`
}
