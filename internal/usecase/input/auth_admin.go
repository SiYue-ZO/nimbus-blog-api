package input

type ChangePassword struct {
	ID             int64
	OldPassword    string
	NewPassword    string
	ClearResetFlag bool
}

type UpdateAdminProfile struct {
	ID             int64
	Nickname       string
	Specialization string
}
