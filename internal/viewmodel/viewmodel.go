// Package viewmodel contains display-ready data contracts for AsymmFlow screens
// and agent consumers.
package viewmodel

// ListVM represents a paginated list ready for display.
type ListVM[T any] struct {
	Items      []T    `json:"items"`
	TotalCount int    `json:"totalCount"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	HasMore    bool   `json:"hasMore"`
	SortBy     string `json:"sortBy,omitempty"`
	SortDesc   bool   `json:"sortDesc,omitempty"`
}

// SummaryCard represents a dashboard summary card.
type SummaryCard struct {
	Label    string  `json:"label"`
	Value    string  `json:"value"`
	Subtext  string  `json:"subtext,omitempty"`
	Trend    string  `json:"trend,omitempty"`
	TrendPct float64 `json:"trendPct,omitempty"`
	Color    string  `json:"color,omitempty"`
}

// ActionButton represents a contextual action.
type ActionButton struct {
	Label   string `json:"label"`
	Action  string `json:"action"`
	Icon    string `json:"icon,omitempty"`
	Variant string `json:"variant,omitempty"`
	Enabled bool   `json:"enabled"`
}

// BreadcrumbItem represents navigation context.
type BreadcrumbItem struct {
	Label string `json:"label"`
	Path  string `json:"path,omitempty"`
}

// FormField represents a form input configuration.
type FormField struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Value       any      `json:"value"`
	Required    bool     `json:"required"`
	Disabled    bool     `json:"disabled,omitempty"`
	Options     []Option `json:"options,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"`
	Validation  string   `json:"validation,omitempty"`
}

// Option represents a select/dropdown option.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
