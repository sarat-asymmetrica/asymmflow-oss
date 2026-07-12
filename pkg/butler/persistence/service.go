// Package persistence contains Butler chat persistence helpers.
package persistence

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	butler "ph_holdings_app/pkg/butler"
)

var actionTypeAlias = map[string]string{
	"create_offer_draft":      "create",
	"create_offer":            "create",
	"create_quotation":        "create",
	"createoffer":             "create",
	"create_order":            "create",
	"createorder":             "create",
	"create_purchase_order":   "create",
	"createfollowup":          "create",
	"create_followup":         "create",
	"create_follow_up":        "create",
	"create_followup_task":    "create",
	"create_follow_up_task":   "create",
	"create_stock_adjustment": "create",
	"create_stock":            "create",
	"create_stockadjustment":  "create",
	"create_customer":         "create",
	"createcustomer":          "create",
	"create_supplier":         "create",
	"createsupplier":          "create",
	"daily_briefing":          "daily_briefing",
	"dailybriefing":           "daily_briefing",
	"clarify":                 "clarify",
	"clarification":           "clarify",
	"choose":                  "clarify",
	"open":                    "open",
	"navigate":                "navigate",
	"analyze":                 "analyze",
	"fetch":                   "fetch",
}

var actionTargetAlias = map[string]string{
	"offer_draft":       "offer",
	"offerdraft":        "offer",
	"po":                "purchase_order",
	"purchaseorder":     "purchase_order",
	"purchase-orders":   "purchase_order",
	"purchase_orders":   "purchase_order",
	"quotation":         "offer",
	"quote":             "offer",
	"quotations":        "offer",
	"follow-up":         "follow_up",
	"followup":          "follow_up",
	"customercontact":   "customer_contact",
	"customer_contact":  "customer_contact",
	"customer contact":  "customer_contact",
	"customer-contacts": "customer_contact",
	"suppliercontact":   "supplier_contact",
	"supplier_contact":  "supplier_contact",
	"supplier contact":  "supplier_contact",
	"supplier-contacts": "supplier_contact",
	"followup_task":     "follow_up",
	"follow_up_task":    "follow_up",
	"followup task":     "follow_up",
	"follow-up task":    "follow_up",
	"costingsheet":      "costing_sheet",
	"costings":          "costing_sheet",
	"costingsheets":     "costing_sheet",
	"supplierinvoice":   "supplier_invoice",
	"supplier_invoice":  "supplier_invoice",
	"stockadjustment":   "stock_adjustment",
	"stock-adjustment":  "stock_adjustment",
	"stockadjust":       "stock_adjustment",
	"stockadjustments":  "stock_adjustment",
	"stock_adjustments": "stock_adjustment",
	"customer":          "customer",
	"customers":         "customer",
	"supplier":          "supplier",
	"suppliers":         "supplier",
	"purchase_order":    "purchase_order",
	"offer":             "offer",
	"daily briefing":    "daily_briefing",
	"dailybriefing":     "daily_briefing",
	"tasks":             "follow_up",
	"task":              "follow_up",
	"follow up":         "follow_up",
	"opportunities":     "opportunity",
	"contacts":          "contact",
}

func TruncateTitle(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}

func NormalizeAssistantMessageContent(content string, actions []butler.ButlerAction) string {
	trimmed := strings.TrimSpace(content)
	if trimmed != "" {
		return trimmed
	}

	if summary := SummarizeActions(actions); summary != "" {
		return summary
	}

	return "I processed your request, but I do not have a replay-safe text response for this turn."
}

func SummarizeActions(actions []butler.ButlerAction) string {
	normalized := NormalizeActionList(actions)
	if len(normalized) == 0 {
		return ""
	}

	labels := make([]string, 0, len(normalized))
	seen := make(map[string]struct{})
	for _, action := range normalized {
		label := strings.TrimSpace(action.Label)
		if label == "" {
			label = InferActionLabel(action.Type, action.Target)
		}
		if label == "" {
			continue
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		labels = append(labels, label)
		if len(labels) == 3 {
			break
		}
	}

	if len(labels) == 0 {
		return ""
	}
	if len(labels) == 1 {
		return fmt.Sprintf("I prepared a suggested action: %s.", labels[0])
	}

	return fmt.Sprintf("I prepared %d suggested actions: %s.", len(normalized), strings.Join(labels, "; "))
}

func ExtractStoredActions(msg butler.ChatMessage) []butler.ButlerAction {
	if strings.TrimSpace(msg.ActionMetadata) != "" {
		var persisted struct {
			Actions []struct {
				Type   string `json:"type"`
				Target string `json:"target"`
				Label  string `json:"label"`
				Data   any    `json:"data"`
			} `json:"actions"`
		}
		if err := json.Unmarshal([]byte(msg.ActionMetadata), &persisted); err == nil && len(persisted.Actions) > 0 {
			actions := make([]butler.ButlerAction, 0, len(persisted.Actions))
			for _, action := range persisted.Actions {
				actions = append(actions, butler.ButlerAction{
					Type:   action.Type,
					Target: action.Target,
					Label:  action.Label,
					Data:   action.Data,
				})
			}
			return NormalizeActionList(actions)
		}
	}

	if strings.TrimSpace(msg.ActionType) != "" || strings.TrimSpace(msg.ActionTarget) != "" || strings.TrimSpace(msg.ActionLabel) != "" {
		return NormalizeActionList([]butler.ButlerAction{{
			Type:   msg.ActionType,
			Target: msg.ActionTarget,
			Label:  msg.ActionLabel,
		}})
	}

	return nil
}

func ReplaySafeMessageContent(msg butler.ChatMessage) string {
	trimmed := strings.TrimSpace(msg.Content)
	if trimmed != "" {
		return trimmed
	}
	if msg.Role != "assistant" {
		return ""
	}
	return NormalizeAssistantMessageContent("", ExtractStoredActions(msg))
}

func BuildReplayMessage(msg butler.ChatMessage) (map[string]any, bool) {
	content := ReplaySafeMessageContent(msg)
	if content == "" {
		return nil, false
	}

	return map[string]any{
		"role":    msg.Role,
		"content": content,
	}, true
}

func NormalizeActionList(actions []butler.ButlerAction) []butler.ButlerAction {
	normalized := make([]butler.ButlerAction, 0, len(actions))
	for _, action := range actions {
		nType, nTarget := normalizedActionTypeTarget(action.Type, action.Target)
		label := strings.TrimSpace(action.Label)
		if label == "" {
			label = InferActionLabel(nType, nTarget)
		}
		normalized = append(normalized, butler.ButlerAction{
			Type:   nType,
			Target: nTarget,
			Label:  label,
			Data:   action.Data,
		})
	}
	return normalized
}

func InferActionLabel(actionType, target string) string {
	if actionType == "" {
		return "Action"
	}

	prettyTarget := strings.ReplaceAll(target, "_", " ")
	if prettyTarget == "" {
		return fmt.Sprintf("%s action", strings.ToUpper(string(actionType[0]))+actionType[1:])
	}

	return fmt.Sprintf("%s %s", strings.ToUpper(string(actionType[0]))+actionType[1:], strings.TrimSpace(prettyTarget))
}

func normalizedActionTypeTarget(actionType, target string) (string, string) {
	normalizedType := NormalizeActionType(actionType)
	normalizedTarget := NormalizeActionTarget(target)

	if normalizedType == "create" && (normalizedTarget == "quotation" || normalizedTarget == "quote") {
		normalizedTarget = "offer"
	}

	if normalizedType == "daily_briefing" {
		normalizedTarget = "daily_briefing"
	}

	return normalizedType, normalizedTarget
}

func NormalizeActionType(actionType string) string {
	value := NormalizeActionToken(actionType)
	if alias, ok := actionTypeAlias[value]; ok {
		return alias
	}
	return value
}

func NormalizeActionTarget(target string) string {
	value := NormalizeActionToken(target)
	if alias, ok := actionTargetAlias[value]; ok {
		return alias
	}
	return value
}

func NormalizeActionToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "\t", "_")
	value = strings.ReplaceAll(value, "\n", "_")
	for strings.Contains(value, "__") {
		value = strings.ReplaceAll(value, "__", "_")
	}
	return strings.Trim(value, "_")
}
