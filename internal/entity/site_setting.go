package entity

import "time"

const (
	SettingTypeString  = "string"
	SettingTypeNumber  = "number"
	SettingTypeBoolean = "boolean"
	SettingTypeJSON    = "json"
)

type SiteSetting struct {
	ID           int64
	SettingKey   string
	SettingValue *string
	SettingType  string
	Description  *string
	IsPublic     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
