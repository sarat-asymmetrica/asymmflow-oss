package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAssistantMessageContentFallsBackToActionSummary(t *testing.T) {
	content := normalizeAssistantMessageContent("", []ButlerAction{{
		Type:   "fetch",
		Target: "follow_up",
		Label:  "Fetch tasks assigned to Jamie",
	}})

	require.Equal(t, "I prepared a suggested action: Fetch tasks assigned to Jamie.", content)
}

func TestGetConversationMessagesHydratesLegacyAssistantActionOnlyMessages(t *testing.T) {
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Conversation{}, &ChatMessage{}))

	conversationID := uuid.New().String()
	require.NoError(t, app.db.Create(&Conversation{
		Base:      Base{ID: conversationID, CreatedAt: time.Now(), UpdatedAt: time.Now(), CreatedBy: app.currentUserID},
		Title:     "Butler regression",
		IsActive:  true,
		LastMsgAt: time.Now(),
	}).Error)

	require.NoError(t, app.db.Create(&ChatMessage{
		Base:           Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		ConversationID: conversationID,
		Role:           "assistant",
		Content:        "",
		MessageType:    "action_suggestion",
		ActionType:     "fetch",
		ActionTarget:   "follow_up",
		ActionLabel:    "Fetch tasks assigned to Jamie",
	}).Error)

	messages, err := app.GetConversationMessages(conversationID)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "I prepared a suggested action: Fetch tasks assigned to Jamie.", messages[0].Content)
}

func TestBuildReplayMessageSkipsEmptyNonAssistantMessages(t *testing.T) {
	replayMsg, ok := buildReplayMessage(ChatMessage{
		Role:    "user",
		Content: "   ",
	})

	require.False(t, ok)
	require.Nil(t, replayMsg)
}

func TestParseMistralResponseActionOnlyProducesReplaySafeMessage(t *testing.T) {
	response := parseMistralResponse(`[ACTIONS][{"type":"fetch","target":"follow_up","label":"Fetch tasks assigned to Jamie"}][/ACTIONS]`, map[string]any{}, Intent{})

	require.Equal(t, "I prepared a suggested action: Fetch tasks assigned to Jamie.", response.Message)
	require.Len(t, response.Actions, 1)
}

func TestChatWithButlerPersistent_TaskCreationUsesGroundedFastPath(t *testing.T) {
	app := setupHybridFeatureTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Conversation{}, &ChatMessage{}))
	seedCurrentAdminContext(t, app, "Jordan")

	jamie := createEmployeeForTest(t, app, "Jamie Wong", "Jamie", "Sales", "Coordinator")

	resp, err := app.ChatWithButlerPersistent("", "can you create a task for Jamie to follow up on National Petroleum lead. make sure he gets the notifications")
	require.NoError(t, err)
	require.Contains(t, resp.Response, "Created the task")

	var tasks []TaskItem
	require.NoError(t, app.db.Where("assignee_employee_id = ?", jamie.ID).Find(&tasks).Error)
	require.Len(t, tasks, 1)

	var messages []ChatMessage
	require.NoError(t, app.db.Where("conversation_id = ?", resp.ConversationID).Order("created_at ASC").Find(&messages).Error)
	require.Len(t, messages, 2)
	require.Equal(t, "user", messages[0].Role)
	require.Equal(t, "assistant", messages[1].Role)
	require.Contains(t, messages[1].Content, "Created the task")
}
