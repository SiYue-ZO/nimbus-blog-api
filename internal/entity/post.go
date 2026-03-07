package entity

import "time"

const (
	PostStatusDraft     = "draft"
	PostStatusPublished = "published"
	PostStatusArchived  = "archived"
)

type Post struct {
	ID              int64
	Title           string
	Slug            string
	Excerpt         *string
	Content         string
	FeaturedImage   *string
	AuthorID        int64
	CategoryID      int64
	Status          string
	ReadTime        *string
	Views           int32
	Likes           int32
	IsFeatured      bool
	MetaTitle       *string
	MetaDescription *string
	PublishedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
