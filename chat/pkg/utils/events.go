package utils

import (
	"chat/internal/models"
	"chat/pkg/redis"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const MessageTypeEvent = "new_message"

type MessageEvent struct {
	EventType      string    `json:"event_type"`
	ID             string    `json:"id"`
	ChatID         string    `json:"chat_id"`
	UserID         string    `json:"user_id"`
	Content        string    `json:"content"`
	Type           string    `json:"type"`
	CreatedAt      time.Time `json:"created_at"`
	ReplyToMessage string    `json:"reply_to_message"`
	Participants   []string  `json:"participants"`
}

func PublishMessage(chatID uuid.UUID, msg *models.Message, pIDs []string) error {
	ctx := context.Background()
	event := MessageEvent{
		EventType:    MessageTypeEvent,
		ID:           msg.ID.String(),
		ChatID:       chatID.String(),
		Content:      msg.Content,
		Type:         msg.Type,
		CreatedAt:    msg.CreatedAt,
		Participants: pIDs,
	}
	if msg.UserID != nil {
		event.UserID = msg.UserID.String()
	} else {
		event.UserID = ""
	}
	if msg.ReplyToMessage != nil {
		event.ReplyToMessage = msg.ReplyToMessage.String()
	} else {
		event.ReplyToMessage = ""
	}

	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	fmt.Println("Опубликовано message:", event, "для пользователей", event.Participants)
	return redis.ChatRedis.Publish(ctx, "chat:message:events", bytes).Err()
}
