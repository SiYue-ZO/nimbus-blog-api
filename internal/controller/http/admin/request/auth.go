package request

type Login struct {
	Username     string `json:"username" validate:"required,min=3,max=32"`
	Password     string `json:"password" validate:"required,min=1,max=20"`
	OTPCode      string `json:"otp_code" validate:"omitempty,len=6,numeric"`
	RecoveryCode string `json:"recovery_code" validate:"omitempty,min=8,max=64"`
}

type ResetPassword struct {
	Username    string `json:"username" validate:"required,min=3,max=32"`
	OldPassword string `json:"old_password" validate:"required,min=1,max=20"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=20,nefield=OldPassword"`
}

type ChangePassword struct {
	OldPassword string `json:"old_password" validate:"required,min=1,max=20"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=20,nefield=OldPassword"`
}

type TwoFASetup struct{}

type TwoFAVerify struct {
	SetupID string `json:"setup_id" validate:"required,min=16,max=128"`
	Code    string `json:"code" validate:"required,len=6,numeric"`
}

type UpdateProfile struct {
	Nickname       string `json:"nickname" validate:"required,min=1,max=100"`
	Specialization string `json:"specialization" validate:"required,min=1,max=200"`
}

type TwoFAAuth struct {
	Code         string `json:"code" validate:"omitempty,len=6,numeric"`
	RecoveryCode string `json:"recovery_code" validate:"omitempty,min=8,max=64"`
}
