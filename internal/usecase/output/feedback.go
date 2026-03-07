package output

import "time"

type FeedbackDetail struct {
	ID        int64
	Name      string
	Email     string
	Type      string
	Subject   string
	Message   string
	Status    string
	IPAddress *string
	UserAgent *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
