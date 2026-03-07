package output

import "time"

type BasePost struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	Excerpt       string     `json:"excerpt"`
	FeaturedImage string     `json:"featured_image"`
	AuthorID      int64      `json:"author_id"`
	Status        string     `json:"status"`
	ReadTime      string     `json:"read_time"`
	Views         int32      `json:"views"`
	IsFeatured    bool       `json:"is_featured"`
	PublishedAt   *time.Time `json:"published_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type AuthorInfo struct {
	ID             int64  `json:"id"`
	Nickname       string `json:"nickname"`
	Specialization string `json:"specialization"`
}

type LikeInfo struct {
	Liked *bool
	Likes int32
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
	BaseCategory
	PostCount int32     `json:"post_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TagDetail struct {
	BaseTag
	PostCount int32     `json:"post_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PostSummary struct {
	BasePost
	Author   AuthorInfo   `json:"author"`
	Like     LikeInfo     `json:"like"`
	Category BaseCategory `json:"category"`
	Tags     []BaseTag    `json:"tags"`
}

type PostDetail struct {
	BasePost
	Author          AuthorInfo   `json:"author"`
	Like            LikeInfo     `json:"like"`
	Category        BaseCategory `json:"category"`
	Tags            []BaseTag    `json:"tags"`
	Content         string       `json:"content"`
	MetaTitle       string       `json:"meta_title"`
	MetaDescription string       `json:"meta_description"`
}
