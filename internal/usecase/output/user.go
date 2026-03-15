package output

import "time"

type UserDetail struct {
	ID              int64
	Name            string
	Email           *string
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

type UserProfile struct {
	Name    string
	Avatar  string
	Bio     string
	Status  string
	BlogURL *string
	UserProfileExtended
}

type UserProfileExtended struct {
	Email  *string
	Region *string
}
