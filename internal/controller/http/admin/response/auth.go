package response

type Login struct {
	RequiresReset bool `json:"requires_reset"`
	OTPRequired   bool `json:"otp_required"`
}

type TwoFASetupStart struct {
	SetupID     string `json:"setup_id"`
	Secret      string `json:"secret"`
	QRCodeImage string `json:"qrcode_image_base64"`
}

type TwoFAVerifyResult struct {
	Enabled         bool     `json:"enabled"`
	ReloginRequired bool     `json:"relogin_required"`
	RecoveryCodes   []string `json:"recovery_codes"`
}

type AdminProfile struct {
	Nickname       string `json:"nickname"`
	Specialization string `json:"specialization"`
	TwoFAEnabled   bool   `json:"twofa_enabled"`
}

type ResetRecoveryCodes struct {
	RecoveryCodes []string `json:"recovery_codes"`
}
