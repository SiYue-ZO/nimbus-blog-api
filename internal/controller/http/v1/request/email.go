package request

type SendCode struct {
	Email     string `json:"email" validate:"required,email"`
	CaptchaID string `json:"captcha_id" validate:"required,min=1,max=64"`
	Captcha   string `json:"captcha" validate:"required,min=4,max=10"`
}
