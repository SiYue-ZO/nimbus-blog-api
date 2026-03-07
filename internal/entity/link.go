package entity

import "time"

const (
	LinkStatusActive   = "active"
	LinkStatusInactive = "inactive"
)

type Link struct {
	ID          int64
	Name        string
	URL         string
	Description *string
	Logo        *string
	SortOrder   int32
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
