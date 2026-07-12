// Package shared contains reusable ViewModel primitives for screen composition.
package shared

import vm "ph_holdings_app/internal/viewmodel"

// TableVM represents a sortable, paginated data table.
type TableVM struct {
	Columns    []TableColumn `json:"columns"`
	Rows       []TableRow    `json:"rows"`
	TotalRows  int           `json:"totalRows"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	SortColumn string        `json:"sortColumn"`
	SortDesc   bool          `json:"sortDesc"`
	Filters    []TableFilter `json:"filters,omitempty"`
}

// TableColumn describes a display column.
type TableColumn struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Sortable bool   `json:"sortable"`
	Width    string `json:"width,omitempty"`
	Align    string `json:"align,omitempty"`
	Currency string `json:"currency,omitempty"`
}

// TableRow is a display-ready table row.
type TableRow struct {
	ID      string            `json:"id"`
	Fields  map[string]any    `json:"fields"`
	Actions []vm.ActionButton `json:"actions,omitempty"`
	Status  string            `json:"status,omitempty"`
}

// TableFilter describes a filter control for a table column.
type TableFilter struct {
	Column  string      `json:"column"`
	Type    string      `json:"type"`
	Value   any         `json:"value,omitempty"`
	Options []vm.Option `json:"options,omitempty"`
}

// DashboardVM represents a dashboard screen layout.
type DashboardVM struct {
	Title       string            `json:"title"`
	Subtitle    string            `json:"subtitle,omitempty"`
	Cards       []vm.SummaryCard  `json:"cards"`
	Actions     []vm.ActionButton `json:"actions,omitempty"`
	LastUpdated string            `json:"lastUpdated"`
}

// StatusBadgeVM represents an entity status.
type StatusBadgeVM struct {
	Label   string `json:"label"`
	Color   string `json:"color"`
	Icon    string `json:"icon,omitempty"`
	Tooltip string `json:"tooltip,omitempty"`
}
