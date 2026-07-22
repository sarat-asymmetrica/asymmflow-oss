package india

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

//go:embed data/states.json
var statesJSON []byte

// State is one GST state/UT code entry.
type State struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

var (
	statesOnce   sync.Once
	statesByCode map[string]string
	allStates    []State
)

// loadStates parses data/states.json once. Code 25 (the old standalone
// "Daman and Diu") is deliberately absent: it was merged into 26 (Dadra and
// Nagar Haveli and Daman and Diu) in 2020 and is no longer an active GST
// state code — this is data, not an oversight.
func loadStates() {
	statesOnce.Do(func() {
		var states []State
		if err := json.Unmarshal(statesJSON, &states); err != nil {
			// The embedded file is repo-controlled and covered by tests;
			// a parse failure here means the data file itself is broken.
			panic("india: malformed embedded states.json: " + err.Error())
		}
		statesByCode = make(map[string]string, len(states))
		for _, s := range states {
			statesByCode[s.Code] = s.Name
		}
		allStates = states
	})
}

// StateName returns the registered name for a 2-digit GST state code and
// whether the code is known.
func StateName(code string) (string, bool) {
	loadStates()
	name, ok := statesByCode[normalizeStateCode(code)]
	return name, ok
}

// ValidStateCode reports whether code is a known 2-digit GST state/UT code.
func ValidStateCode(code string) bool {
	loadStates()
	_, ok := statesByCode[normalizeStateCode(code)]
	return ok
}

// AllStates returns every registered state/UT, sorted by code.
func AllStates() []State {
	loadStates()
	out := append([]State(nil), allStates...)
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

func normalizeStateCode(code string) string {
	return strings.TrimSpace(code)
}
