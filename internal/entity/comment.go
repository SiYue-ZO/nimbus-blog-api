package entity

import "time"

const (
	CommentStatusPending  = "pending"
	CommentStatusApproved = "approved"
	CommentStatusRejected = "rejected"
	CommentStatusSpam     = "spam"
)

type Comment struct {
	ID        int64
	PostID    int64
	ParentID  *int64
	UserID    int64
	Content   string
	Status    string
	Likes     int32
	IPAddress *string
	UserAgent *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
