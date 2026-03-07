package request

type CreatePost struct {
	Title         string  `json:"title" validate:"required,min=1,max=200"`
	Slug          string  `json:"slug" validate:"required,min=1,max=200"`
	Excerpt       *string `json:"excerpt" validate:"omitempty,max=500"`
	Content       string  `json:"content" validate:"required,min=1"`
	FeaturedImage *string `json:"featured_image" validate:"omitempty,max=1000"`
	AuthorID      int64   `json:"author_id" validate:"required,gte=1"`
	CategoryID    int64   `json:"category_id" validate:"required,gte=1"`
	TagIDs        []int64 `json:"tag_ids" validate:"omitempty,dive,gte=1"`
	Status        string  `json:"status" validate:"required,oneof=draft published archived"`
	IsFeatured    bool    `json:"is_featured"`
}

type UpdatePost struct {
	ID            int64   `json:"id" validate:"required,gte=1"`
	Title         string  `json:"title" validate:"omitempty,max=200"`
	Slug          string  `json:"slug" validate:"omitempty,max=200"`
	Excerpt       *string `json:"excerpt" validate:"omitempty,max=500"`
	Content       string  `json:"content"`
	FeaturedImage *string `json:"featured_image" validate:"omitempty,max=1000"`
	AuthorID      int64   `json:"author_id" validate:"omitempty,gte=1"`
	CategoryID    int64   `json:"category_id" validate:"omitempty,gte=1"`
	TagIDs        []int64 `json:"tag_ids" validate:"omitempty,dive,gte=1"`
	Status        string  `json:"status" validate:"omitempty,oneof=draft published archived"`
	IsFeatured    bool    `json:"is_featured"`
}

type CreateCategory struct {
	Name string `json:"name" validate:"required,min=1,max=50"`
	Slug string `json:"slug" validate:"required,min=1,max=50"`
}

type UpdateCategory struct {
	ID   int64  `json:"id" validate:"required,gte=1"`
	Name string `json:"name" validate:"required,min=1,max=50"`
	Slug string `json:"slug" validate:"required,min=1,max=50"`
}

type CreateTag struct {
	Name string `json:"name" validate:"required,min=1,max=50"`
	Slug string `json:"slug" validate:"required,min=1,max=50"`
}

type UpdateTag struct {
	ID   int64  `json:"id" validate:"required,gte=1"`
	Name string `json:"name" validate:"required,min=1,max=50"`
	Slug string `json:"slug" validate:"required,min=1,max=50"`
}

type GenerateSlug struct {
	Input string `json:"input" validate:"required,min=1,max=200"`
}
