package input

type UpsertSiteSetting struct {
	SettingKey   string
	SettingValue *string
	SettingType  string
	Description  *string
	IsPublic     bool
}
