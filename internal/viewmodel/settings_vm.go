package viewmodel

// SettingsVM is the display contract for application settings.
type SettingsVM struct {
	Sections []SettingsSectionVM `json:"sections"`
}

// SettingsSectionVM groups related settings fields.
type SettingsSectionVM struct {
	Title  string      `json:"title"`
	Icon   string      `json:"icon,omitempty"`
	Fields []FormField `json:"fields"`
}
