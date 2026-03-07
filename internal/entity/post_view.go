package entity

import "time"

type PostView struct {
	ID        int64
	PostID    int64
	IPAddress string
	UserAgent *string
	Referer   *string
	ViewedAt  time.Time
}
