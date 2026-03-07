package request

type UpdateUserStatus struct {
	ID     int64  `json:"id" validate:"required,gte=1"`
	Status string `json:"status" validate:"required,oneof=active disabled"`
}
