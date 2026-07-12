// Package butler contains display-ready ViewModels for Butler intelligence screens.
package butler

import (
	vm "ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/internal/viewmodel/shared"
)

// ChatVM is the display contract for the Butler chat screen.
type ChatVM struct {
	ConversationID   string            `json:"conversationId"`
	Title            string            `json:"title"`
	Messages         []ChatMessageVM   `json:"messages"`
	SuggestedActions []vm.ActionButton `json:"suggestedActions"`
	InputPlaceholder string            `json:"inputPlaceholder"`
	IsTyping         bool              `json:"isTyping"`
}

// ChatMessageVM is a display-ready chat message.
type ChatMessageVM struct {
	ID        string            `json:"id"`
	Role      string            `json:"role"`
	Content   string            `json:"content"`
	Timestamp string            `json:"timestamp"`
	Actions   []vm.ActionButton `json:"actions,omitempty"`
}

// ConversationListVM is the display contract for Butler conversation history.
type ConversationListVM struct {
	Conversations []ConversationPreviewVM `json:"conversations"`
	ActiveID      string                  `json:"activeId,omitempty"`
	Actions       []vm.ActionButton       `json:"actions"`
}

// ConversationPreviewVM displays one Butler conversation in a list.
type ConversationPreviewVM struct {
	ID            string               `json:"id"`
	Title         string               `json:"title"`
	Preview       string               `json:"preview"`
	LastMessageAt string               `json:"lastMessageAt"`
	Status        shared.StatusBadgeVM `json:"status"`
}

// DailyBriefingVM is the display contract for the daily Butler briefing.
type DailyBriefingVM struct {
	Title            string            `json:"title"`
	Date             string            `json:"date"`
	Highlights       []vm.SummaryCard  `json:"highlights"`
	Risks            []ButlerInsightVM `json:"risks"`
	Priorities       []ButlerInsightVM `json:"priorities"`
	SuggestedActions []vm.ActionButton `json:"suggestedActions"`
}

// PredictionVM displays one prediction with confidence treatment.
type PredictionVM struct {
	ID              string               `json:"id"`
	Subject         string               `json:"subject"`
	Prediction      string               `json:"prediction"`
	Confidence      string               `json:"confidence"`
	ConfidenceBadge shared.StatusBadgeVM `json:"confidenceBadge"`
	ExpectedDate    string               `json:"expectedDate,omitempty"`
	AmountDisplay   string               `json:"amountDisplay,omitempty"`
	Reasoning       []string             `json:"reasoning,omitempty"`
	Actions         []vm.ActionButton    `json:"actions,omitempty"`
}

// ButlerInsightVM is a structured insight card for dashboards and briefings.
type ButlerInsightVM struct {
	ID       string               `json:"id"`
	Title    string               `json:"title"`
	Message  string               `json:"message"`
	Severity shared.StatusBadgeVM `json:"severity"`
	Source   string               `json:"source,omitempty"`
	Metric   string               `json:"metric,omitempty"`
	Actions  []vm.ActionButton    `json:"actions,omitempty"`
}

// InputStateVM describes chat input affordances.
type InputStateVM struct {
	Placeholder string   `json:"placeholder"`
	Disabled    bool     `json:"disabled"`
	Hints       []string `json:"hints,omitempty"`
}
