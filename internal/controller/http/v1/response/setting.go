package response

import "time"

type SiteSettingDetail struct {
	ID           int64     `json:"id"`
	SettingKey   string    `json:"setting_key"`
	SettingValue *string   `json:"setting_value"`
	SettingType  string    `json:"setting_type"`
	Description  *string   `json:"description"`
	IsPublic     bool      `json:"is_public"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
