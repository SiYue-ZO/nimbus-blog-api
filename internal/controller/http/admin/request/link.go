package request

type CreateLink struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	URL         string  `json:"url" validate:"required,http_url"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	Logo        *string `json:"logo" validate:"omitempty,max=1000"`
	SortOrder   int32   `json:"sort_order" validate:"gte=0,lte=9999"`
	Status      string  `json:"status" validate:"required,oneof=active inactive"`
}

type UpdateLink struct {
	ID          int64   `json:"id" validate:"required,gte=1"`
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	URL         string  `json:"url" validate:"required,http_url"`
	Description *string `json:"description" validate:"omitempty,max=500"`
	Logo        *string `json:"logo" validate:"omitempty,max=1000"`
	SortOrder   int32   `json:"sort_order" validate:"gte=0,lte=9999"`
	Status      string  `json:"status" validate:"required,oneof=active inactive"`
}
