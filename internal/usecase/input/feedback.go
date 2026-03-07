package input

type ListFeedbacks struct {
	PageParams
	Sort   *SortParams
	Status StringFilterParam
}

type UpdateFeedback struct {
	ID     int64
	Status string
}

type SubmitFeedback struct {
	Name      string
	Email     string
	Type      string
	Subject   string
	Message   string
	IPAddress *string
	UserAgent *string
}
