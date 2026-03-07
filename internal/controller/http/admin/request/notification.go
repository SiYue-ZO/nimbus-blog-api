package request

type SendNotification struct {
	UserID  int64  `json:"user_id" validate:"required,gte=1"`
	Title   string `json:"title" validate:"required,min=1,max=200"`
	Content string `json:"content" validate:"required,min=1,max=5000"`
}
