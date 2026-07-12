package butler

import (
	"testing"
	"time"

	gormbutler "ph_holdings_app/pkg/butler"
	shareddomain "ph_holdings_app/pkg/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatMessageRoundtrip(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	original := gormbutler.ChatMessage{
		Base:           shareddomain.Base{ID: "msg-1", CreatedAt: now, UpdatedAt: now, CreatedBy: "codex"},
		ConversationID: "conversation-1",
		Role:           "assistant",
		Content:        "Here is the cash position.",
		TokensUsed:     88,
		MessageType:    "chat",
		ActionType:     "open_screen",
		ActionTarget:   "finance",
		ActionLabel:    "Open Finance",
		ActionData:     `{"screen":"finance"}`,
		ActionStatus:   "pending",
		ActionMetadata: `{"source":"test"}`,
	}

	p, err := ChatMessageToProto(original)
	require.NoError(t, err)
	back, err := ChatMessageFromProto(*p)
	require.NoError(t, err)

	assert.Equal(t, original.ID, back.ID)
	assert.Equal(t, original.ConversationID, back.ConversationID)
	assert.Equal(t, original.Role, back.Role)
	assert.Equal(t, original.Content, back.Content)
	assert.Equal(t, original.TokensUsed, back.TokensUsed)
	assert.Equal(t, original.MessageType, back.MessageType)
	assert.Equal(t, original.ActionType, back.ActionType)
	assert.Equal(t, original.ActionTarget, back.ActionTarget)
	assert.Equal(t, original.ActionLabel, back.ActionLabel)
	assert.Equal(t, original.ActionData, back.ActionData)
	assert.Equal(t, original.ActionStatus, back.ActionStatus)
	assert.Equal(t, original.ActionMetadata, back.ActionMetadata)
}
