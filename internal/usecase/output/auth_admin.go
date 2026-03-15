package output

import "time"

type AdminDetail struct {
	ID                int64
	Username          string
	PasswordHash      string
	Nickname          string
	Specialization    string
	MustResetPassword bool
	TwoFactorSecret   *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type AdminProfile struct {
	Nickname       string
	Specialization string
	TwoFAEnabled   bool
}

type TwoFASetupStart struct {
	SetupID  string
	Secret   string
	QRBase64 string
}

type TwoFAVerifyResult struct {
	Enabled       bool
	RecoveryCodes []string
}
