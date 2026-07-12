package ui_alchemy

import (
	"fmt"
	"reflect"
)

const (
	PHI     = 1.618033988749895
	PHI_INV = 0.618033988749895
)

// FieldType represents the semantic type of a field
type FieldType string

const (
	FieldText     FieldType = "text"
	FieldNumber   FieldType = "number"
	FieldDate     FieldType = "date"
	FieldBoolean  FieldType = "boolean"
	FieldCurrency FieldType = "currency"
	FieldEnum     FieldType = "enum"
)

// LayoutField represents a single field in the UI
type LayoutField struct {
	Name        string    `json:"name"`
	Label       string    `json:"label"`
	Type        FieldType `json:"type"`
	Width       string    `json:"width"` // "100%", "50%", "33%" based on φ
	Order       int       `json:"order"`
	Regime      int       `json:"regime"` // 1, 2, 3
	Placeholder string    `json:"placeholder"`
}

// FormLayout represents a complete form structure
type FormLayout struct {
	EntityName  string        `json:"entity_name"`
	Seed        int64         `json:"seed"`
	DigitalRoot int           `json:"digital_root"`
	Fields      []LayoutField `json:"fields"`
	Theme       ThemeHints    `json:"theme"`
}

// ComponentType defines the type of UI component
type ComponentType string

const (
	ComponentSurvivalGarden     ComponentType = "survival_garden"
	ComponentOpportunityMandala ComponentType = "opportunity_mandala"
	ComponentServiceRhythm      ComponentType = "service_rhythm"
	ComponentFlowCard           ComponentType = "flow_card"
	ComponentQuickCapture       ComponentType = "quick_capture"
	ComponentZenNav             ComponentType = "zen_nav"
	ComponentContextPointer     ComponentType = "context_pointer"
	ComponentToast              ComponentType = "toast"
	ComponentModal              ComponentType = "modal"
	ComponentConfirmDialog      ComponentType = "confirm_dialog"
	ComponentStatusBadge        ComponentType = "status_badge"
	ComponentHealthBar          ComponentType = "health_bar"
	ComponentMetricCard         ComponentType = "metric_card"
	ComponentToggleButton       ComponentType = "toggle_button"
	ComponentDataTable          ComponentType = "data_table"
	ComponentTaskFlow           ComponentType = "task_flow"
	ComponentSearchInput        ComponentType = "search_input"
	ComponentDatePicker         ComponentType = "date_picker"
	ComponentDropdown           ComponentType = "dropdown"
	ComponentLoadingSpinner     ComponentType = "loading_spinner"
	ComponentEmptyState         ComponentType = "empty_state"
	ComponentForm               ComponentType = "form"
	ComponentButlerInsight      ComponentType = "butler_insight" // New Intelligence Component
)

// ContextVector represents the current state of the business/user context
type ContextVector struct {
	TimeOfDay string  // "morning", "afternoon", "evening"
	FlowRate  float64 // MB/s or Activity Level
	Urgency   float64 // 0.0 - 1.0 (Risk Level)
}

// Component represents a generic UI component in the layout
type Component struct {
	ID       string         `json:"id"`
	Type     ComponentType  `json:"type"`
	Data     map[string]any `json:"data"`
	GridArea string         `json:"grid_area"` // CSS Grid Area
	Regime   int            `json:"regime"`    // 1, 2, 3
}

// ScreenLayout represents a full page structure
type ScreenLayout struct {
	ID           string       `json:"id"`
	Title        string       `json:"title"`
	Type         string       `json:"type"` // "dashboard", "list", "detail"
	Components   []Component  `json:"components"`
	Theme        ThemeHints   `json:"theme"`
	GridTemplate string       `json:"grid_template"` // CSS Grid Template
	Regime       VisualRegime `json:"regime"`        // The visual state for the renderer
}

type ThemeHints struct {
	PrimaryColor    string `json:"primary_color"`
	AccentColor     string `json:"accent_color"`
	BackgroundColor string `json:"background_color"`
}

// GenerateScreen creates a screen layout based on the requested screen ID and Context
// NOW DETERMINISTICALLY DRIVEN BY CONTEXT VECTOR
func GenerateScreen(screenID string, ctx ContextVector) ScreenLayout {
	seed := calculateSeed(screenID)
	dr := digitalRoot(int(seed))
	theme := generateThemeHints(dr)

	// Calculate Visual Regime for this context
	visualRegime := GetVisualRegime(ctx)

	components := make([]Component, 0)
	gridTemplate := ""

	switch screenID {
	case "dashboard":
		// DYNAMIC REGIME INJECTION
		// The layout morphs based on the Time of Day and Flow Rate

		if ctx.TimeOfDay == "morning" {
			// MORNING REGIME: Focus on Planning & Insights (Butler + Tasks)
			// Layout: Butler Insight Top, Tasks Left, Garden Right (Monitoring)
			gridTemplate = `"insight insight insight" "tasks tasks garden" "tasks tasks garden"`

			components = append(components, Component{
				ID:       "butler_insight",
				Type:     ComponentButlerInsight,
				GridArea: "insight",
				Regime:   1,
				Data: map[string]any{
					"message":   "Good morning. 3 High-Priority RFQs require review. System flow is nominal.",
					"sentiment": "calm",
				},
			})
		} else if ctx.FlowRate > 50.0 { // High Activity
			// FLOW REGIME: Focus on Real-time (Garden + Signals)
			// Layout: Garden Dominates (Wave), Signals Below
			gridTemplate = `"garden garden garden" "garden garden garden" "signals signals signals"`
		} else {
			// STANDARD REGIME (Balanced)
			gridTemplate = `"garden garden tasks" "garden garden tasks" "signals signals signals"`
		}

		// Core Components (Always present, but might shift position)
		components = append(components, Component{
			ID:       "survival_garden",
			Type:     ComponentSurvivalGarden,
			GridArea: "garden",
			Regime:   3,
			Data: map[string]any{
				"runwayMonths": 6.2,
				"burnRate":     11000,
				"flowRate":     ctx.FlowRate, // Pass backend flow rate to physics
			},
		})

		if ctx.TimeOfDay == "morning" || ctx.FlowRate <= 50.0 {
			components = append(components, Component{
				ID:       "todays_flow",
				Type:     ComponentTaskFlow,
				GridArea: "tasks",
				Regime:   1,
				Data: map[string]any{
					"tasks": []map[string]any{
						{"title": "Approve GSC Quote", "time": "10:00 AM", "subtitle": "Phase 3 Expansion", "color": "#ef4444"},
						{"title": "Review NPC Invoice", "time": "11:30 AM", "subtitle": "Maintenance Contract", "color": "#fbbf24"},
						{"title": "Call Supplier", "time": "2:00 PM", "subtitle": "Delay check", "color": "#15803d"},
					},
				},
			})
		}

		if ctx.TimeOfDay != "morning" {
			components = append(components, Component{
				ID:       "live_signals",
				Type:     ComponentDataTable,
				GridArea: "signals",
				Regime:   2,
				Data: map[string]any{
					"title":   "Live Signals",
					"columns": []string{"Type", "Message", "Confidence", "Time"},
					"data": []map[string]any{
						{"Type": "📢 RFQ", "Message": "New Request from DPC", "Confidence": "98%", "Time": "Just now"},
						{"Type": "💰 PAY", "Message": "Payment Received (NPC)", "Confidence": "100%", "Time": "10m ago"},
					},
				},
			})
		}

	case "opportunities":
		gridTemplate = `"mandala list list" "details list list"`
		components = append(components, Component{
			ID:       "opp_mandala",
			Type:     ComponentOpportunityMandala,
			GridArea: "mandala",
			Regime:   1, // Exploration
			Data: map[string]any{
				"winProb": 0.45,
			},
		})
		components = append(components, Component{
			ID:       "opp_list",
			Type:     ComponentDataTable,
			GridArea: "list",
			Regime:   2,
			Data: map[string]any{
				"title":   "Active Pipeline",
				"columns": []string{"Name", "Stage", "Value", "Probability"},
				"data": []map[string]any{
					{"Name": "GSC Phase 3", "Stage": "Quote Sent", "Value": "125,000 BHD", "Probability": "45%"},
					{"Name": "NPC Maint", "Stage": "Negotiation", "Value": "85,000 BHD", "Probability": "32%"},
					{"Name": "DPC Analyzer", "Stage": "Lead", "Value": "12,000 BHD", "Probability": "10%"},
				},
			},
		})

	case "orders": // Service Rhythm screen
		gridTemplate = `"rhythm list" "rhythm list"`
		components = append(components, Component{
			ID:       "service_rhythm",
			Type:     ComponentServiceRhythm,
			GridArea: "rhythm",
			Regime:   3,
			Data: map[string]any{
				"mrr":   4000,
				"costs": 15000,
			},
		})
		components = append(components, Component{
			ID:       "order_list",
			Type:     ComponentDataTable,
			GridArea: "list",
			Regime:   3,
			Data: map[string]any{
				"title":   "Recent Orders",
				"columns": []string{"ID", "Customer", "Status", "Total"},
				"data": []map[string]any{
					{"ID": "ORD-2025-001", "Customer": "GSC", "Status": "Delivered", "Total": "45,000 BHD"},
					{"ID": "ORD-2025-002", "Customer": "NPC", "Status": "Processing", "Total": "12,500 BHD"},
				},
			},
		})

	case "butler":
		gridTemplate = `"logs logs logs"`
		components = append(components, Component{
			ID:       "file_logs",
			Type:     ComponentDataTable,
			GridArea: "logs",
			Regime:   3,
			Data: map[string]any{
				"title":   "Butler File Watcher Logs",
				"columns": []string{"Time", "Event", "Path", "Size"},
				"data": []map[string]any{
					// Real data will be piped in via frontend store
					{"Time": "10:45:01", "Event": "CREATE", "Path": "/invoices/INV-101.pdf", "Size": "45KB"},
					{"Time": "10:45:02", "Event": "PROCESS", "Path": "Parsed Invoice #101", "Size": "-"},
				},
			},
		})

	case "customers":
		gridTemplate = `"list list list"`
		components = append(components, Component{
			ID:       "customer_list",
			Type:     ComponentDataTable,
			GridArea: "list",
			Regime:   3,
			Data: map[string]any{
				"title":   "Customer Directory",
				"columns": []string{"ID", "Name", "Grade", "Relation"},
				"data": []map[string]any{
					{"ID": "CUST-001", "Name": "GSC", "Grade": "A", "Relation": "15 Years"},
					{"ID": "CUST-002", "Name": "NPC", "Grade": "A", "Relation": "12 Years"},
				},
			},
		})

	case "settings":
		gridTemplate = `"config config config"`
		components = append(components, Component{
			ID:       "config_form",
			Type:     ComponentForm,
			GridArea: "config",
			Regime:   3,
			Data: map[string]any{
				// This mimics the FormLayout structure but wrapped in a Component
				"entity_name": "System Configuration",
				"fields": []LayoutField{
					{Name: "watcher_path", Label: "Watch Folder", Type: "text", Width: "100%"},
					{Name: "api_key", Label: "API Key", Type: "text", Width: "61.8%"},
					{Name: "theme", Label: "Theme Mode", Type: "enum", Width: "38.2%"},
				},
			},
		})

	case "costing":
		gridTemplate = `"sheet sheet sheet"`
		components = append(components, Component{
			ID:       "active_costing_sheet",
			Type:     ComponentType("costing_sheet"), // Cast string to ComponentType
			GridArea: "sheet",
			Regime:   2,
			Data: map[string]any{
				"currency": "BHD",
				"markup":   1.2,
				"items": []map[string]any{
					{"description": "Industrial Pump X200", "quantity": 2, "unitCost": 450, "margin": 25},
					{"description": "Installation Service", "quantity": 1, "unitCost": 150, "margin": 30},
					{"description": "Logistics & Handling", "quantity": 1, "unitCost": 50, "margin": 15},
				},
			},
		})
	}

	return ScreenLayout{
		ID:           screenID,
		Title:        prettify(screenID),
		Type:         "generated",
		Components:   components,
		Theme:        theme,
		GridTemplate: gridTemplate,
		Regime:       visualRegime,
	}
}

// GenerateForm creates a math-based layout from a struct instance
func GenerateForm(entity any) FormLayout {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := t.Name()
	seed := calculateSeed(name)
	dr := digitalRoot(int(seed))

	fields := make([]LayoutField, 0)

	// Iterate fields and apply Phi-based sizing
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}

		// Calculate semantic width based on Digital Root and index
		width := calculateWidth(i, dr)

		// Determine type
		fType := determineType(f.Type)

		fields = append(fields, LayoutField{
			Name:        jsonTag,
			Label:       prettify(f.Name),
			Type:        fType,
			Width:       width,
			Order:       i,
			Regime:      (i % 3) + 1, // Cycle through regimes
			Placeholder: fmt.Sprintf("Enter %s...", prettify(f.Name)),
		})
	}

	return FormLayout{
		EntityName:  name,
		Seed:        seed,
		DigitalRoot: dr,
		Fields:      fields,
		Theme:       generateThemeHints(dr),
	}
}

// Math helpers
func calculateSeed(s string) int64 {
	var sum int64
	for _, c := range s {
		sum += int64(c)
	}
	return sum
}

func digitalRoot(n int) int {
	if n == 0 {
		return 0
	}
	return 1 + (n-1)%9
}

func calculateWidth(index int, dr int) string {
	// Use φ pattern: 100%, 61.8%, 38.2%
	// Digital root offsets the pattern
	pattern := (index + dr) % 3

	switch pattern {
	case 0:
		return "100%" // Full width
	case 1:
		return "61.8%" // Major section
	case 2:
		return "38.2%" // Minor section (sidebar/detail)
	}
	return "100%"
}

func determineType(t reflect.Type) FieldType {
	switch t.Kind() {
	case reflect.String:
		return FieldText
	case reflect.Int, reflect.Int64, reflect.Float64:
		return FieldNumber
	case reflect.Bool:
		return FieldBoolean
	case reflect.Struct:
		if t.Name() == "Time" {
			return FieldDate
		}
	}
	return FieldText
}

func generateThemeHints(dr int) ThemeHints {
	// Simple palette based on digital root
	// Added BackgroundColor (Wabi-Sabi Rice Paper)
	paper := "#fdfbf7"

	palettes := []ThemeHints{
		{PrimaryColor: "#1c1c1c", AccentColor: "#fbbf24", BackgroundColor: paper}, // 9/0: Void/Gold
		{PrimaryColor: "#2d4a6f", AccentColor: "#00d4ff", BackgroundColor: paper}, // 1: Deep Blue/Cyan
		{PrimaryColor: "#15803d", AccentColor: "#86efac", BackgroundColor: paper}, // 2: Forest/Green
		{PrimaryColor: "#b91c1c", AccentColor: "#fca5a5", BackgroundColor: paper}, // 3: Red/Pink
		{PrimaryColor: "#7c2d12", AccentColor: "#fdba74", BackgroundColor: paper}, // 4: Earth/Orange
		{PrimaryColor: "#4c1d95", AccentColor: "#c4b5fd", BackgroundColor: paper}, // 5: Violet/Lavender
		{PrimaryColor: "#0f766e", AccentColor: "#5eead4", BackgroundColor: paper}, // 6: Teal/Aqua
		{PrimaryColor: "#be185d", AccentColor: "#f9a8d4", BackgroundColor: paper}, // 7: Pink/Rose
		{PrimaryColor: "#1e293b", AccentColor: "#94a3b8", BackgroundColor: paper}, // 8: Slate/Grey
	}
	return palettes[dr%9]
}

func prettify(s string) string {
	// Very basic camelCase to Title Case
	// In production, use a proper library
	return s
}
