package request

type UpdateCommentStatus struct {
	ID     int64  `json:"id" validate:"required,gte=1"`
	Status string `json:"status" validate:"required,oneof=approved rejected spam"`
}
