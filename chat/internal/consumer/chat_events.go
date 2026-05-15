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
			userID, err := uuid.Parse(event.UserID)
			if err != nil {
				log.Printf("Invalid userID in read_update: %v", err)
				return
			}
			chatID, err := uuid.Parse(event.ChatID)
			if err != nil {
				log.Printf("Invalid chatID in read_update: %v", err)
				return
			}

			// Batch UPDATE — обновляем только указанные сообщения, которые ещё не прочитаны этим пользователем
			if len(event.MessageIDs) > 0 {
				db.Model(&models.Message{}).
					Where("id IN ? AND chat_id = ? AND (user_id IS NULL OR user_id != ?) AND ((NOT (? = ANY(read_by))) OR read_by IS NULL)",
						event.MessageIDs, chatID, userID, userID).
					Update("read_by", gorm.Expr("array_append(read_by, ?)", userID))
			}

			// Получаем участников для рассылки ack
			var participants []models.Participant
			db.Where("chat_id = ?", chatID).Find(&participants)
			var pIds []string
			for _, p := range participants {
				pIds = append(pIds, p.UserID.String())
			}

			ack := map[string]interface{}{
				"event":        "read_ack",
				"chat_id":      event.ChatID,
				"user_id":      event.UserID,
				"participants": pIds,
				"message_ids":  event.MessageIDs,
			}
			payload, _ := json.Marshal(ack)

			if err := redis.ChatRedis.Publish(context.Background(), "chat:read_ack:events", payload).Err(); err != nil {
				log.Printf("Failed to publish read_ack event: %v", err)
			}
		}
	})
	if err != nil {
		log.Fatalf("Failed to start consume (ChatEvents): %v", err)
	}
	log.Println("Chat events consumer started")
}
