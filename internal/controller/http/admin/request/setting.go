package request

type UpsertSiteSetting struct {
	SettingKey   string  `json:"setting_key" validate:"required,min=1,max=100"`
	SettingValue *string `json:"setting_value" validate:"omitempty,max=10000"`
	SettingType  string  `json:"setting_type" validate:"required,oneof=string number boolean json"`
	Description  *string `json:"description" validate:"omitempty,max=500"`
	IsPublic     bool    `json:"is_public"`
}
