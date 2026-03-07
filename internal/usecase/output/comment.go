package output

import "time"

type BaseComment struct {
	ID           int64
	PostID       int64
	ParentID     *int64
	UserID       int64
	Content      string
	RepliesCount int32
}

type CommentDetail struct {
	BaseComment
	UserProfile
	Like      LikeInfo
	Status    string
	IPAddress *string
	UserAgent *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CommentBasic struct {
	BaseComment
	UserProfile
	Like      LikeInfo
	CreatedAt time.Time
}
