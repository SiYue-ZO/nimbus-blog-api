package entity

import "time"

type Admin struct {
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
