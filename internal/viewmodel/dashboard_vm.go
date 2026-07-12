package viewmodel

// MainDashboardVM is the display contract for the home dashboard.
type MainDashboardVM struct {
	Greeting         string           `json:"greeting"`
	Date             string           `json:"date"`
	QuickStats       []SummaryCard    `json:"quickStats"`
	RecentActivity   []ActivityItemVM `json:"recentActivity"`
	UpcomingTasks    []TaskItemVM     `json:"upcomingTasks"`
	CashPosition     PanelRefVM       `json:"cashPosition"`
	PipelineSnapshot PanelRefVM       `json:"pipelineSnapshot"`
	Alerts           []AlertVM        `json:"alerts"`
}

// PanelRefVM carries a composed child ViewModel without forcing root-package
// imports back into child packages.
type PanelRefVM struct {
	Type  string `json:"type"`
	Title string `json:"title,omitempty"`
	Data  any    `json:"data"`
}

// ActivityItemVM is a display-ready dashboard activity row.
type ActivityItemVM struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Timestamp   string `json:"timestamp"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	TargetPath  string `json:"targetPath,omitempty"`
}

// TaskItemVM is a display-ready dashboard task row.
type TaskItemVM struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	DueDate  string `json:"dueDate"`
	Priority string `json:"priority"`
	Status   string `json:"status"`
	Owner    string `json:"owner,omitempty"`
}

// AlertVM is a display-ready dashboard alert.
type AlertVM struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Message  string       `json:"message"`
	Severity string       `json:"severity"`
	Icon     string       `json:"icon,omitempty"`
	Action   ActionButton `json:"action,omitempty"`
}
