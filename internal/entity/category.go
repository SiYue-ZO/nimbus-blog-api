package entity

import (
	"time"
)

type Category struct {
	ID        int64
	Name      string
	Slug      string
	PostCount int32
	CreatedAt time.Time
	UpdatedAt time.Time
}
