package response

import "time"

type AuthorInfo struct {
	ID             int64  `json:"id"`
	Nickname       string `json:"nickname"`
	Specialization string `json:"specialization"`
}

type PostSummary struct {
	ID            int64        `json:"id"`
	Title         string       `json:"title"`
	Slug          string       `json:"slug"`
	Excerpt       string       `json:"excerpt"`
	FeaturedImage string       `json:"featured_image"`
	AuthorID      int64        `json:"author_id"`
	Author        AuthorInfo   `json:"author"`
	Status        string       `json:"status"`
	ReadTime      string       `json:"read_time"`
	Views         int32        `json:"views"`
	Likes         int32        `json:"likes"`
	IsFeatured    bool         `json:"is_featured"`
	PublishedAt   *time.Time   `json:"published_at"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	Category      BaseCategory `json:"category"`
	Tags          []BaseTag    `json:"tags"`
}

type PostDetail struct {
	ID              int64        `json:"id"`
	Title           string       `json:"title"`
	Slug            string       `json:"slug"`
	Excerpt         string       `json:"excerpt"`
	FeaturedImage   string       `json:"featured_image"`
	AuthorID        int64        `json:"author_id"`
	Author          AuthorInfo   `json:"author"`
	Status          string       `json:"status"`
	ReadTime        string       `json:"read_time"`
	Views           int32        `json:"views"`
	Likes           int32        `json:"likes"`
	IsFeatured      bool         `json:"is_featured"`
	PublishedAt     *time.Time   `json:"published_at"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
	Category        BaseCategory `json:"category"`
	Tags            []BaseTag    `json:"tags"`
	Content         string       `json:"content"`
	MetaTitle       string       `json:"meta_title"`
	MetaDescription string       `json:"meta_description"`
}

type BaseCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type BaseTag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CategoryDetail struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	PostCount int32     `json:"post_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TagDetail struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	PostCount int32     `json:"post_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatePost struct {
	ID int64 `json:"id"`
}

type CreateTag struct {
	ID int64 `json:"id"`
}

type CreateCategory struct {
	ID int64 `json:"id"`
}

type GenerateSlug struct {
	Slug string `json:"slug"`
}
