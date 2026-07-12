// Package butler contains the Butler intelligence domain model.
package butler

import (
	"time"

	shareddomain "ph_holdings_app/pkg/domain"
)

type Base = shareddomain.Base

// ButlerResponse represents AI response with suggested actions.
type ButlerResponse struct {
	Message    string                 `json:"message"`
	Actions    []ButlerAction         `json:"actions"`
	Confidence float64                `json:"confidence"`
	Context    map[string]any         `json:"context"`
	Metadata   ButlerResponseMetadata `json:"metadata"`
}

type ButlerResponseMetadata struct {
	UsedBackend       string         `json:"used_backend"`
	RequestedModel    string         `json:"requested_model"`
	UsedModel         string         `json:"used_model"`
	FallbackReason    string         `json:"fallback_reason"`
	FinanceDataAccess bool           `json:"finance_data_access"`
	ContextMode       string         `json:"context_mode"`
	DataCoverage      []string       `json:"data_coverage"`
	EntityResolution  map[string]any `json:"entity_resolution"`
	GeneratedAt       string         `json:"generated_at"`
	Error             string         `json:"error"`
}

// ButlerAction represents a suggested action from Butler.
type ButlerAction struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	Label  string `json:"label"`
	Data   any    `json:"data"`
}

// Intent represents classified user intent.
type Intent struct {
	RawQuery      string
	Domain        string
	EntityName    string
	PersonName    string
	ReferenceKind string
	Confidence    float64
	Keywords      []string
	NeedsScraper  bool
	IsComplex     bool
}

type ButlerResolvedEntity struct {
	EntityType        string
	EntityID          string
	DisplayName       string
	Confidence        float64
	MatchReason       string
	RelatedCustomerID string
	RelatedCustomer   string
	Ambiguous         bool
	Alternatives      []map[string]any
}

type PredictionRecord struct {
	Base
	// Composite index for payment prediction lookups: WHERE customer_id = ? AND grade = ?
	// Used by: Payment intelligence engine, predictive Butler, survival analysis
	// Priority 1 = CustomerID (high cardinality), Priority 2 = Grade (low cardinality)
	CustomerID    string  `gorm:"index:idx_prediction_customer_grade,priority:1;size:36" json:"customer_id"`
	CustomerName  string  `json:"customer_name"`
	Grade         string  `gorm:"index:idx_prediction_customer_grade,priority:2;size:10" json:"grade"`
	PredictedDays int     `json:"predicted_days"`
	Confidence    float64 `json:"confidence"`
	R1            float64 `json:"r1"`
	R2            float64 `json:"r2"`
	R3            float64 `json:"r3"`
}

type WinProbabilityPrediction struct {
	Base
	OfferID              string  `gorm:"index;size:36" json:"offer_id"`
	PredictedProbability float64 `json:"predicted_probability"`
}

type DiscountRecommendationRecord struct {
	Base
	OfferID             string  `gorm:"index;size:36" json:"offer_id"`
	RecommendedDiscount float64 `json:"recommended_discount"`
}

type CustomerSnapshot struct {
	Base
	CustomerID string  `gorm:"index;size:36" json:"customer_id"`
	OrderValue float64 `json:"order_value"`
}

type ActualOutcome struct {
	Base
	CustomerID string `gorm:"index;size:36" json:"customer_id"`
	WasPaid    bool   `json:"was_paid"`
}

type PaymentPredictionAccuracy struct {
	Base
	CustomerID  string `gorm:"index;size:36" json:"customer_id"`
	WasAccurate bool   `json:"was_accurate"`
}

type Conversation struct {
	Base
	Title     string    `gorm:"size:255" json:"title"`
	Summary   string    `gorm:"type:varchar(2000)" json:"summary"`
	IsActive  bool      `gorm:"index;default:true" json:"is_active"`
	LastMsgAt time.Time `gorm:"index;autoUpdateTime" json:"last_msg_at"`
}

func (Conversation) TableName() string { return "conversations" }

type ChatMessage struct {
	Base
	ConversationID string `gorm:"index;size:36;constraint:OnDelete:CASCADE,OnUpdate:CASCADE;foreignKey:ConversationID;references:ID" json:"conversation_id"`
	Role           string `gorm:"size:20;check:role IN ('user','assistant','system')" json:"role"` // "user" or "assistant"
	Content        string `gorm:"type:text" json:"content"`
	TokensUsed     int    `gorm:"check:tokens_used >= 0" json:"tokens_used"`
	MessageType    string `gorm:"size:50;default:'chat'" json:"message_type"`
	ActionType     string `gorm:"size:50" json:"action_type"`
	ActionTarget   string `gorm:"size:100" json:"action_target"`
	ActionLabel    string `gorm:"size:100" json:"action_label"`
	ActionData     string `gorm:"type:text" json:"action_data"`
	ActionStatus   string `gorm:"size:50;default:'none'" json:"action_status"`
	ActionMetadata string `gorm:"type:text" json:"action_metadata"`
}

func (ChatMessage) TableName() string { return "chat_messages" }
