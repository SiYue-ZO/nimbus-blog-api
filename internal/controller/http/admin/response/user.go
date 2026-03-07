package response

import "time"

type UserDetail struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Email           *string   `json:"email"`
	Avatar          string    `json:"avatar"`
	Bio             string    `json:"bio"`
	Status          string    `json:"status"`
	EmailVerified   bool      `json:"email_verified"`
	Region          *string   `json:"region"`
	BlogURL         *string   `json:"blog_url"`
	AuthProvider    *string   `json:"auth_provider"`
	AuthOpenid      *string   `json:"auth_openid"`
	ShowFullProfile bool      `json:"show_full_profile"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type UserProfile struct {
	Name    string  `json:"name"`
	Avatar  string  `json:"avatar"`
	Bio     string  `json:"bio"`
	Status  string  `json:"status"`
	BlogURL *string `json:"blog_url"`
	Email   *string `json:"email"`
	Region  *string `json:"region"`
}
