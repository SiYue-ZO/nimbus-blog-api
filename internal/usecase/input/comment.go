package input

type ListComments struct {
	PageParams
	Sort   *SortParams
	Status StringFilterParam
}

type SubmitComment struct {
	PostID    int64
	ParentID  *int64
	UserID    int64
	Content   string
	IPAddress *string
	UserAgent *string
}
