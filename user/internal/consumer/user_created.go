package consumer

import (
	"encoding/json"
	"fmt"
	"log"
	"user/internal/models"
	"user/pkg/rabbitmq"
	"user/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func StartUserEventsConsumer(db *gorm.DB) {
	tx := db.Begin()

	err := rabbitmq.Consume("user.events", func(body []byte) {
		var event struct {
			UserID string `json:"user_id"`
			Action string `json:"action"`
		}
		if err := json.Unmarshal(body, &event); err != nil {
			fmt.Printf("Invalid event JSON: %v", err)
			return
		}
		if event.Action != "user_created" {
			return // игнорируем другие события
		}
		userID := uuid.MustParse(event.UserID)

		name := "user_" + event.UserID[:8]
		profile := models.Profile{
			ID:        userID,
			Username:  name,
			FullName:  name,
			AvatarURL: utils.RandomAvatar(),
		}
		settings := models.Settings{
			ProfileID: userID,
		}

		if err := tx.Create(&profile).Error; err != nil {
			log.Printf("Ошибка. Не удалось создать профиль %s: %v", userID, err)
			tx.Rollback()
			return
		}
		if err := tx.Create(&settings).Error; err != nil {
			log.Printf("Ошибка. Не удалось добавить настройки %s: %v", userID, err)
			tx.Rollback()
			return
		} else {
			tx.Commit()
		}
	})
	if err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}
	log.Println("User events consumer started")

}
