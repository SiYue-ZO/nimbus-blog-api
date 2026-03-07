package request

type SubmitComment struct {
	ParentID *int64 `json:"parent_id"`
	Content  string `json:"content" validate:"required,min=1,max=5000"`
}
