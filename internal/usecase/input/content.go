package input

type CreateCategory struct {
	Name string
	Slug string
}

type UpdateCategory struct {
	ID   int64
	Name string
	Slug string
}

type CreateTag struct {
	Name string
	Slug string
}

type ListCategories struct {
	PageParams
	Keyword *KeywordParams
	Sort    *SortParams
}

type UpdateTag struct {
	ID   int64
	Name string
	Slug string
}

type ListTags struct {
	PageParams
	Keyword *KeywordParams
	Sort    *SortParams
}

type CreatePost struct {
	Title         string
	Slug          string
	Excerpt       *string
	Content       string
	FeaturedImage *string
	AuthorID      int64
	CategoryID    int64
	TagIDs        []int64
	Status        string
	IsFeatured    bool
}

type UpdatePost struct {
	ID            int64
	Title         string
	Slug          string
	Excerpt       *string
	Content       string
	FeaturedImage *string
	AuthorID      int64
	CategoryID    int64
	TagIDs        []int64
	Status        string
	IsFeatured    bool
}

type ListPosts struct {
	PageParams
	Keyword    *KeywordParams
	Sort       *SortParams
	CategoryID IntFilterParam
	TagID      IntFilterParam
	Status     StringFilterParam
	IsFeatured BoolFilterParam
}

type ListPublicPosts struct {
	PageParams
	Keyword    *KeywordParams
	Sort       *SortParams
	CategoryID IntFilterParam
	TagID      IntFilterParam
}
