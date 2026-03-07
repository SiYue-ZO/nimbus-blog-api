package output

import "time"

type SiteSettingDetail struct {
	ID           int64
	SettingKey   string
	SettingValue *string
	SettingType  string
	Description  *string
	IsPublic     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
