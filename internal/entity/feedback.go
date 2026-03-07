package entity

import "time"

const (
	FeedbackTypeGeneral = "general"
	FeedbackTypeBug     = "bug"
	FeedbackTypeFeature = "feature"
	FeedbackTypeUI      = "ui"
)

const (
	FeedbackStatusPending    = "pending"
	FeedbackStatusProcessing = "processing"
	FeedbackStatusResolved   = "resolved"
	FeedbackStatusClosed     = "closed"
)

type Feedback struct {
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
