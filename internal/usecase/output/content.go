package output

import "time"

type BasePost struct {
	ID            int64
	Title         string
	Slug          string
	Excerpt       string
	FeaturedImage string
	AuthorID      int64
	Status        string
	ReadTime      string
	Views         int32
	IsFeatured    bool
	PublishedAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AuthorInfo struct {
	ID             int64
	Nickname       string
	Specialization string
}

type LikeInfo struct {
	Liked *bool
	Likes int32
}

type BaseCategory struct {
	ID   int64
	Name string
	Slug string
}

type BaseTag struct {
	ID   int64
	Name string
	Slug string
}

type CategoryDetail struct {
	BaseCategory
	PostCount int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TagDetail struct {
	BaseTag
	PostCount int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PostSummary struct {
	BasePost
	Author   AuthorInfo
	Like     LikeInfo
	Category BaseCategory
	Tags     []BaseTag
}

type PostDetail struct {
	BasePost
	Author          AuthorInfo
	Like            LikeInfo
	Category        BaseCategory
	Tags            []BaseTag
	Content         string
	MetaTitle       string
	MetaDescription string
}
