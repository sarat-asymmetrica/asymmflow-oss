package posting

type CoverageReport struct {
	Rows         []CoverageRow `json:"rows"`
	Total        int64         `json:"total"`
	Linked       int64         `json:"linked"`
	Missing      int64         `json:"missing"`
	DraftEntries int64         `json:"draft_entries"`
	IsComplete   bool          `json:"is_complete"`
}

type CoverageRow struct {
	SourceType   string `json:"source_type"`
	Label        string `json:"label"`
	Total        int64  `json:"total"`
	Linked       int64  `json:"linked"`
	Missing      int64  `json:"missing"`
	DraftEntries int64  `json:"draft_entries"`
	IsComplete   bool   `json:"is_complete"`
}

func BuildCoverageReport(rows []CoverageRow) CoverageReport {
	report := CoverageReport{Rows: rows}
	for i := range report.Rows {
		report.Rows[i].Missing = report.Rows[i].Total - report.Rows[i].Linked
		if report.Rows[i].Missing < 0 {
			report.Rows[i].Missing = 0
		}
		report.Rows[i].IsComplete = report.Rows[i].Missing == 0
		report.Total += report.Rows[i].Total
		report.Linked += report.Rows[i].Linked
		report.Missing += report.Rows[i].Missing
		report.DraftEntries += report.Rows[i].DraftEntries
	}
	report.IsComplete = report.Missing == 0
	return report
}
