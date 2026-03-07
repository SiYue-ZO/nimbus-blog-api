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
	SetupID  string `json:"setup_id"`
	Secret   string `json:"secret"`
	QRBase64 string `json:"qr_base64"`
}

type TwoFAVerifyResult struct {
	Enabled       bool     `json:"enabled"`
	RecoveryCodes []string `json:"recovery_codes"`
}
