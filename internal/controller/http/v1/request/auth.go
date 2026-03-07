package request

type Register struct {
	Username string `json:"username" validate:"required,min=2,max=32"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=20"`
	Code     string `json:"code" validate:"required,len=6,numeric"`
}

type Login struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=1,max=20"`
	CaptchaID string `json:"captcha_id" validate:"required,min=1,max=64"`
	Captcha   string `json:"captcha" validate:"required,len=6,numeric"`
}

type ChangePassword struct {
	OldPassword string `json:"old_password" validate:"required,min=1,max=20"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=20"`
}

type ForgotPassword struct {
	Email       string `json:"email" validate:"required,email"`
	Code        string `json:"code" validate:"required,len=6,numeric"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=20"`
}
