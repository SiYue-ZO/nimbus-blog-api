package response

import "time"

type UserProfile struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Email           *string   `json:"email"`
	Avatar          string    `json:"avatar"`
	Bio             string    `json:"bio"`
	Status          string    `json:"status"`
	EmailVerified   bool      `json:"email_verified"`
	Region          *string   `json:"region"`
	BlogURL         *string   `json:"blog_url"`
	ShowFullProfile bool      `json:"show_full_profile"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
