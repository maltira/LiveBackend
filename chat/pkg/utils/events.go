package utils

import (
	"chat/internal/models"
	"context"
	"encoding/json"
	"user/pkg/redis"

	"github.com/google/uuid"
)

const MessageTypeEvent = "new_message"

type MessageEvent struct {
	EventType    string   `json:"event_type"`
	ChatID       string   `json:"chat_id"`
	UserID       string   `json:"user_id"`
	Content      string   `json:"content"`
	Type         string   `json:"type"`
	CreatedAt    string   `json:"created_at"`
	Participants []string `json:"participants"`
}

func PublishMessage(chatID uuid.UUID, msg *models.Message, pIDs []string) error {
	ctx := context.Background()
	event := MessageEvent{
		EventType:    MessageTypeEvent,
		ChatID:       chatID.String(),
		Content:      msg.Content,
		Type:         msg.Type,
		CreatedAt:    msg.CreatedAt.String(),
		Participants: pIDs,
	}
	if msg.UserID != nil {
		event.UserID = msg.UserID.String()
	} else {
		event.UserID = ""
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return redis.UserRedis.Publish(ctx, "chat:message:events", bytes).Err()
}
