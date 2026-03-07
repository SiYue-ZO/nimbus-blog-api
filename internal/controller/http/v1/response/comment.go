package response

import "time"

type CommentBasic struct {
	ID           int64              `json:"id"`
	PostID       int64              `json:"post_id"`
	ParentID     *int64             `json:"parent_id"`
	UserID       int64              `json:"user_id"`
	Content      string             `json:"content"`
	Like         LikeInfo           `json:"like"`
	RepliesCount int32              `json:"replies_count"`
	UserProfile  CommentUserProfile `json:"user_profile"`
	CreatedAt    time.Time          `json:"created_at"`
}

type CommentUserProfile struct {
	Name    string  `json:"name"`
	Avatar  string  `json:"avatar"`
	Bio     string  `json:"bio"`
	Status  string  `json:"status"`
	BlogURL *string `json:"blog_url"`
	Email   *string `json:"email"`
	Region  *string `json:"region"`
}
