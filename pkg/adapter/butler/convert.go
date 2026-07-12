// Package butler converts Butler models to and from generated Proto messages.
package butler

import (
	"encoding/json"
	"fmt"
	"strings"

	"ph_holdings_app/pkg/adapter"
	gormbutler "ph_holdings_app/pkg/butler"
	protobutler "ph_holdings_app/schemas/go/butler"
	commonproto "ph_holdings_app/schemas/go/common"
	protocrm "ph_holdings_app/schemas/go/crm"

	capnp "capnproto.org/go/capnp/v3"
)

func newMessage() (*capnp.Message, *capnp.Segment, error) {
	return capnp.NewMessage(capnp.SingleSegment(nil))
}

func setBase(seg *capnp.Segment, setter func(commonproto.Base) error, base gormbutler.Base) error {
	pb, err := adapter.BaseToProto(seg, base)
	if err != nil {
		return err
	}
	return setter(pb)
}

func textList(seg *capnp.Segment, values []string) (capnp.TextList, error) {
	l, err := capnp.NewTextList(seg, int32(len(values)))
	if err != nil {
		return capnp.TextList{}, err
	}
	for i, value := range values {
		if err := l.Set(i, value); err != nil {
			return capnp.TextList{}, err
		}
	}
	return l, nil
}

func keyValues(seg *capnp.Segment, values map[string]any) (commonproto.KeyValue_List, error) {
	l, err := commonproto.NewKeyValue_List(seg, int32(len(values)))
	if err != nil {
		return commonproto.KeyValue_List{}, err
	}
	i := 0
	for key, value := range values {
		kv, err := commonproto.NewKeyValue(seg)
		if err != nil {
			return commonproto.KeyValue_List{}, err
		}
		if err := kv.SetKey(key); err != nil {
			return commonproto.KeyValue_List{}, err
		}
		if err := kv.SetValue(valueString(value)); err != nil {
			return commonproto.KeyValue_List{}, err
		}
		if err := l.Set(i, kv); err != nil {
			return commonproto.KeyValue_List{}, err
		}
		i++
	}
	return l, nil
}

func keyValueSlice(seg *capnp.Segment, values []map[string]any) (commonproto.KeyValue_List, error) {
	flat := make(map[string]any)
	for i, m := range values {
		for key, value := range m {
			flat[fmt.Sprintf("%d.%s", i, key)] = value
		}
	}
	return keyValues(seg, flat)
}

func valueString(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	data, err := json.Marshal(value)
	if err == nil {
		return string(data)
	}
	return fmt.Sprint(value)
}

func grade(value string) protocrm.CustomerGrade {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "A":
		return protocrm.CustomerGrade_a
	case "B":
		return protocrm.CustomerGrade_b
	case "C":
		return protocrm.CustomerGrade_c
	case "D":
		return protocrm.CustomerGrade_d
	default:
		return protocrm.CustomerGrade_unknown
	}
}

func gradeText(value protocrm.CustomerGrade) string {
	switch value {
	case protocrm.CustomerGrade_a:
		return "A"
	case protocrm.CustomerGrade_b:
		return "B"
	case protocrm.CustomerGrade_c:
		return "C"
	case protocrm.CustomerGrade_d:
		return "D"
	default:
		return ""
	}
}

func chatRole(role string) protobutler.ChatRole {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "assistant":
		return protobutler.ChatRole_assistant
	case "system":
		return protobutler.ChatRole_system
	default:
		return protobutler.ChatRole_user
	}
}

func chatRoleText(role protobutler.ChatRole) string {
	switch role {
	case protobutler.ChatRole_assistant:
		return "assistant"
	case protobutler.ChatRole_system:
		return "system"
	default:
		return "user"
	}
}

func actionStatus(status string) protobutler.ActionStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return protobutler.ActionStatus_pending
	case "executed", "complete", "completed":
		return protobutler.ActionStatus_executed
	case "failed":
		return protobutler.ActionStatus_failed
	case "dismissed":
		return protobutler.ActionStatus_dismissed
	default:
		return protobutler.ActionStatus_none
	}
}

func actionStatusText(status protobutler.ActionStatus) string {
	switch status {
	case protobutler.ActionStatus_pending:
		return "pending"
	case protobutler.ActionStatus_executed:
		return "executed"
	case protobutler.ActionStatus_failed:
		return "failed"
	case protobutler.ActionStatus_dismissed:
		return "dismissed"
	default:
		return "none"
	}
}

func ButlerResponseToProto(m gormbutler.ButlerResponse) (*protobutler.ButlerResponse, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewButlerResponse(seg)
	if err != nil {
		return nil, err
	}
	if err := p.SetMessage_(m.Message); err != nil {
		return nil, err
	}
	p.SetConfidence(m.Confidence)
	actions, err := protobutler.NewButlerAction_List(seg, int32(len(m.Actions)))
	if err != nil {
		return nil, err
	}
	for i, action := range m.Actions {
		pa, err := populateButlerAction(seg, action)
		if err != nil {
			return nil, err
		}
		if err := actions.Set(i, pa); err != nil {
			return nil, err
		}
	}
	if err := p.SetActions(actions); err != nil {
		return nil, err
	}
	ctx, err := keyValues(seg, m.Context)
	if err != nil {
		return nil, err
	}
	if err := p.SetContext(ctx); err != nil {
		return nil, err
	}
	meta, err := populateMetadata(seg, m.Metadata)
	if err != nil {
		return nil, err
	}
	if err := p.SetMetadata(meta); err != nil {
		return nil, err
	}
	return &p, nil
}

func ButlerResponseFromProto(p protobutler.ButlerResponse) (gormbutler.ButlerResponse, error) {
	m := gormbutler.ButlerResponse{}
	var err error
	m.Message, err = p.Message_()
	if err != nil {
		return m, err
	}
	m.Confidence = p.Confidence()
	return m, nil
}

func populateMetadata(seg *capnp.Segment, m gormbutler.ButlerResponseMetadata) (protobutler.ButlerResponseMetadata, error) {
	p, err := protobutler.NewButlerResponseMetadata(seg)
	if err != nil {
		return protobutler.ButlerResponseMetadata{}, err
	}
	for _, err := range []error{p.SetUsedBackend(m.UsedBackend), p.SetRequestedModel(m.RequestedModel), p.SetUsedModel(m.UsedModel), p.SetFallbackReason(m.FallbackReason), p.SetContextMode(m.ContextMode), p.SetGeneratedAt(m.GeneratedAt), p.SetError(m.Error)} {
		if err != nil {
			return protobutler.ButlerResponseMetadata{}, err
		}
	}
	p.SetFinanceDataAccess(m.FinanceDataAccess)
	coverage, err := textList(seg, m.DataCoverage)
	if err != nil {
		return protobutler.ButlerResponseMetadata{}, err
	}
	if err := p.SetDataCoverage(coverage); err != nil {
		return protobutler.ButlerResponseMetadata{}, err
	}
	resolution, err := keyValues(seg, m.EntityResolution)
	if err != nil {
		return protobutler.ButlerResponseMetadata{}, err
	}
	if err := p.SetEntityResolution(resolution); err != nil {
		return protobutler.ButlerResponseMetadata{}, err
	}
	return p, nil
}

func ButlerActionToProto(m gormbutler.ButlerAction) (*protobutler.ButlerAction, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := populateButlerAction(seg, m)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func populateButlerAction(seg *capnp.Segment, m gormbutler.ButlerAction) (protobutler.ButlerAction, error) {
	p, err := protobutler.NewButlerAction(seg)
	if err != nil {
		return protobutler.ButlerAction{}, err
	}
	for _, err := range []error{p.SetType(m.Type), p.SetTarget(m.Target), p.SetLabel(m.Label)} {
		if err != nil {
			return protobutler.ButlerAction{}, err
		}
	}
	data, err := keyValues(seg, map[string]any{"value": m.Data})
	if err != nil {
		return protobutler.ButlerAction{}, err
	}
	if err := p.SetData(data); err != nil {
		return protobutler.ButlerAction{}, err
	}
	return p, nil
}

func ButlerActionFromProto(p protobutler.ButlerAction) (gormbutler.ButlerAction, error) {
	m := gormbutler.ButlerAction{}
	m.Type, _ = p.Type()
	m.Target, _ = p.Target()
	m.Label, _ = p.Label()
	return m, nil
}

func IntentToProto(m gormbutler.Intent) (*protobutler.Intent, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewIntent(seg)
	if err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetRawQuery(m.RawQuery), p.SetDomain(m.Domain), p.SetEntityName(m.EntityName), p.SetPersonName(m.PersonName), p.SetReferenceKind(m.ReferenceKind)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetConfidence(m.Confidence)
	p.SetNeedsScraper(m.NeedsScraper)
	p.SetIsComplex(m.IsComplex)
	keywords, err := textList(seg, m.Keywords)
	if err != nil {
		return nil, err
	}
	if err := p.SetKeywords(keywords); err != nil {
		return nil, err
	}
	return &p, nil
}

func IntentFromProto(p protobutler.Intent) (gormbutler.Intent, error) {
	m := gormbutler.Intent{}
	m.RawQuery, _ = p.RawQuery()
	m.Domain, _ = p.Domain()
	m.EntityName, _ = p.EntityName()
	m.Confidence = p.Confidence()
	m.NeedsScraper = p.NeedsScraper()
	m.IsComplex = p.IsComplex()
	return m, nil
}

func ButlerResolvedEntityToProto(m gormbutler.ButlerResolvedEntity) (*protobutler.ButlerResolvedEntity, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewButlerResolvedEntity(seg)
	if err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetEntityType(m.EntityType), p.SetEntityId(m.EntityID), p.SetDisplayName(m.DisplayName), p.SetMatchReason(m.MatchReason), p.SetRelatedCustomerId(m.RelatedCustomerID), p.SetRelatedCustomer(m.RelatedCustomer)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetConfidence(m.Confidence)
	p.SetAmbiguous(m.Ambiguous)
	alternatives, err := keyValueSlice(seg, m.Alternatives)
	if err != nil {
		return nil, err
	}
	if err := p.SetAlternatives(alternatives); err != nil {
		return nil, err
	}
	return &p, nil
}

func ButlerResolvedEntityFromProto(p protobutler.ButlerResolvedEntity) (gormbutler.ButlerResolvedEntity, error) {
	m := gormbutler.ButlerResolvedEntity{}
	m.EntityType, _ = p.EntityType()
	m.EntityID, _ = p.EntityId()
	m.DisplayName, _ = p.DisplayName()
	m.Confidence = p.Confidence()
	m.Ambiguous = p.Ambiguous()
	return m, nil
}

func PredictionRecordToProto(m gormbutler.PredictionRecord) (*protobutler.PredictionRecord, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewPredictionRecord(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetCustomerId(m.CustomerID), p.SetCustomerName(m.CustomerName)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetGrade(grade(m.Grade))
	p.SetPredictedDays(int64(m.PredictedDays))
	p.SetConfidence(m.Confidence)
	p.SetR1(m.R1)
	p.SetR2(m.R2)
	p.SetR3(m.R3)
	return &p, nil
}

func PredictionRecordFromProto(p protobutler.PredictionRecord) (gormbutler.PredictionRecord, error) {
	m := gormbutler.PredictionRecord{}
	m.CustomerID, _ = p.CustomerId()
	m.CustomerName, _ = p.CustomerName()
	m.Grade = gradeText(p.Grade())
	m.PredictedDays = int(p.PredictedDays())
	m.Confidence = p.Confidence()
	return m, nil
}

func WinProbabilityPredictionToProto(m gormbutler.WinProbabilityPrediction) (*protobutler.WinProbabilityPrediction, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewWinProbabilityPrediction(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	if err := p.SetOfferId(m.OfferID); err != nil {
		return nil, err
	}
	p.SetPredictedProbability(m.PredictedProbability)
	return &p, nil
}

func WinProbabilityPredictionFromProto(p protobutler.WinProbabilityPrediction) (gormbutler.WinProbabilityPrediction, error) {
	m := gormbutler.WinProbabilityPrediction{}
	m.OfferID, _ = p.OfferId()
	m.PredictedProbability = p.PredictedProbability()
	return m, nil
}

func DiscountRecommendationRecordToProto(m gormbutler.DiscountRecommendationRecord) (*protobutler.DiscountRecommendationRecord, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewDiscountRecommendationRecord(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	if err := p.SetOfferId(m.OfferID); err != nil {
		return nil, err
	}
	p.SetRecommendedDiscount(m.RecommendedDiscount)
	return &p, nil
}

func DiscountRecommendationRecordFromProto(p protobutler.DiscountRecommendationRecord) (gormbutler.DiscountRecommendationRecord, error) {
	m := gormbutler.DiscountRecommendationRecord{}
	m.OfferID, _ = p.OfferId()
	m.RecommendedDiscount = p.RecommendedDiscount()
	return m, nil
}

func CustomerSnapshotToProto(m gormbutler.CustomerSnapshot) (*protobutler.CustomerSnapshot, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewCustomerSnapshot(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	if err := p.SetCustomerId(m.CustomerID); err != nil {
		return nil, err
	}
	p.SetOrderValue(m.OrderValue)
	return &p, nil
}

func CustomerSnapshotFromProto(p protobutler.CustomerSnapshot) (gormbutler.CustomerSnapshot, error) {
	m := gormbutler.CustomerSnapshot{}
	m.CustomerID, _ = p.CustomerId()
	m.OrderValue = p.OrderValue()
	return m, nil
}

func ActualOutcomeToProto(m gormbutler.ActualOutcome) (*protobutler.ActualOutcome, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewActualOutcome(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	if err := p.SetCustomerId(m.CustomerID); err != nil {
		return nil, err
	}
	p.SetWasPaid(m.WasPaid)
	return &p, nil
}

func ActualOutcomeFromProto(p protobutler.ActualOutcome) (gormbutler.ActualOutcome, error) {
	m := gormbutler.ActualOutcome{}
	m.CustomerID, _ = p.CustomerId()
	m.WasPaid = p.WasPaid()
	return m, nil
}

func PaymentPredictionAccuracyToProto(m gormbutler.PaymentPredictionAccuracy) (*protobutler.PaymentPredictionAccuracy, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewPaymentPredictionAccuracy(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	if err := p.SetCustomerId(m.CustomerID); err != nil {
		return nil, err
	}
	p.SetWasAccurate(m.WasAccurate)
	return &p, nil
}

func PaymentPredictionAccuracyFromProto(p protobutler.PaymentPredictionAccuracy) (gormbutler.PaymentPredictionAccuracy, error) {
	m := gormbutler.PaymentPredictionAccuracy{}
	m.CustomerID, _ = p.CustomerId()
	m.WasAccurate = p.WasAccurate()
	return m, nil
}

func ConversationToProto(m gormbutler.Conversation) (*protobutler.Conversation, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewConversation(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetTitle(m.Title), p.SetSummary(m.Summary), p.SetLastMsgAt(adapter.TimeToText(m.LastMsgAt))} {
		if err != nil {
			return nil, err
		}
	}
	p.SetIsActive(m.IsActive)
	return &p, nil
}

func ConversationFromProto(p protobutler.Conversation) (gormbutler.Conversation, error) {
	base, err := p.Base()
	if err != nil {
		return gormbutler.Conversation{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormbutler.Conversation{}, err
	}
	m := gormbutler.Conversation{Base: sharedBase}
	m.Title, _ = p.Title()
	m.Summary, _ = p.Summary()
	m.IsActive = p.IsActive()
	if s, err := p.LastMsgAt(); err == nil {
		m.LastMsgAt = adapter.TextToTime(s)
	}
	return m, nil
}

func ChatMessageToProto(m gormbutler.ChatMessage) (*protobutler.ChatMessage, error) {
	_, seg, err := newMessage()
	if err != nil {
		return nil, err
	}
	p, err := protobutler.NewChatMessage(seg)
	if err != nil {
		return nil, err
	}
	if err := setBase(seg, p.SetBase, m.Base); err != nil {
		return nil, err
	}
	for _, err := range []error{p.SetConversationId(m.ConversationID), p.SetContent(m.Content), p.SetMessageType(m.MessageType), p.SetActionType(m.ActionType), p.SetActionTarget(m.ActionTarget), p.SetActionLabel(m.ActionLabel), p.SetActionData(m.ActionData), p.SetActionMetadata(m.ActionMetadata)} {
		if err != nil {
			return nil, err
		}
	}
	p.SetRole(chatRole(m.Role))
	p.SetTokensUsed(int64(m.TokensUsed))
	p.SetActionStatus(actionStatus(m.ActionStatus))
	return &p, nil
}

func ChatMessageFromProto(p protobutler.ChatMessage) (gormbutler.ChatMessage, error) {
	base, err := p.Base()
	if err != nil {
		return gormbutler.ChatMessage{}, err
	}
	sharedBase, err := adapter.BaseFromProto(base)
	if err != nil {
		return gormbutler.ChatMessage{}, err
	}
	m := gormbutler.ChatMessage{Base: sharedBase}
	m.ConversationID, _ = p.ConversationId()
	m.Role = chatRoleText(p.Role())
	m.Content, _ = p.Content()
	m.TokensUsed = int(p.TokensUsed())
	m.MessageType, _ = p.MessageType()
	m.ActionType, _ = p.ActionType()
	m.ActionTarget, _ = p.ActionTarget()
	m.ActionLabel, _ = p.ActionLabel()
	m.ActionData, _ = p.ActionData()
	m.ActionStatus = actionStatusText(p.ActionStatus())
	m.ActionMetadata, _ = p.ActionMetadata()
	return m, nil
}
