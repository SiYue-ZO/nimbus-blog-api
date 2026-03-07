package response

import "time"

type LinkDetail struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Description *string   `json:"description"`
	Logo        *string   `json:"logo"`
	SortOrder   int32     `json:"sort_order"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
