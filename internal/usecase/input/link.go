package input

type CreateLink struct {
	Name        string
	URL         string
	Description *string
	Logo        *string
	SortOrder   int32
	Status      string
}

type UpdateLink struct {
	ID          int64
	Name        string
	URL         string
	Description *string
	Logo        *string
	SortOrder   int32
	Status      string
}

type ListLinks struct {
	PageParams
	Keyword *KeywordParams
	Sort    *SortParams
}
