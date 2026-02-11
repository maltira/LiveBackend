package consumer

import (
	"chat/internal/models"
	"chat/pkg/database"
	"chat/pkg/rabbitmq"
	"chat/pkg/redis"
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func StartChatEventsConsumer() {
	db := database.GetDB()

	err := rabbitmq.Consume("chat.events", func(body []byte) {
		var event struct {
			Event      string   `json:"event"`
			UserID     string   `json:"user_id"`
			ChatID     string   `json:"chat_id"`
			MessageIDs []string `json:"message_ids"`
		}
		if err := json.Unmarshal(body, &event); err != nil {
			return
		}

		if event.Event == "read_update" {
			userID, _ := uuid.Parse(event.UserID)
			chatID, _ := uuid.Parse(event.ChatID)

			var messages []models.Message
			db.Where("chat_id = ? AND ((NOT (? = ANY(read_by))) OR read_by IS NULL)", chatID, userID).
				Find(&messages)

			var participants []models.Participant
			var pIds []string
			db.Where("chat_id = ?", chatID).Find(&participants)
			for _, participant := range participants {
				pIds = append(pIds, participant.UserID.String())
			}

			for _, m := range messages {
				m.ReadBy = append(m.ReadBy, userID.String())
				if userID != *m.UserID {
					db.Model(&m).Update("read_by", gorm.Expr("array_append(read_by, ?)", userID))
				}
			}

			ack := map[string]interface{}{
				"event":        "read_ack",
				"chat_id":      event.ChatID,
				"user_id":      event.UserID,
				"participants": pIds,
				"message_ids":  event.MessageIDs,
			}
			payload, _ := json.Marshal(ack)

			err := redis.ChatRedis.Publish(context.Background(), "chat:read_ack:events", payload).Err()
			if err != nil {
				log.Printf("Failed to publish read_ack event: %v", err)
			}
		}
	})
	if err != nil {
		log.Fatalf("Failed to start consume (ChatEvents): %v", err)
	}
	log.Println("Chat events consumer started")
}
