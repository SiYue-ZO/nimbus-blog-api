package request

type SubmitFeedback struct {
	Name    string `json:"name" validate:"required,min=1,max=50"`
	Email   string `json:"email" validate:"required,email"`
	Type    string `json:"type" validate:"required,oneof=general bug feature ui"`
	Subject string `json:"subject" validate:"required,min=1,max=200"`
	Message string `json:"message" validate:"required,min=1,max=5000"`
}
