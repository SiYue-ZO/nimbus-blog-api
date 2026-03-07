package entity

import "time"

const (
	UserStatusActive   = "active"
	UserStatusDisabled = "disabled"
)

type User struct {
	ID              int64
	Name            string
	Email           *string
	PasswordHash    string
	Avatar          string
	Bio             string
	Status          string
	EmailVerified   bool
	Region          *string
	BlogURL         *string
	AuthProvider    *string
	AuthOpenid      *string
	ShowFullProfile bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
