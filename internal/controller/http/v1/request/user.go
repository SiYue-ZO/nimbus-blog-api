package request

type UpdateProfile struct {
	Name            string `json:"name" validate:"omitempty,max=50"`
	Bio             string `json:"bio" validate:"omitempty,max=500"`
	Region          string `json:"region" validate:"omitempty,max=100"`
	BlogURL         string `json:"blog_url" validate:"omitempty,max=500"`
	ShowFullProfile bool   `json:"show_full_profile"`
}
