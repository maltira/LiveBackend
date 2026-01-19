package consumer

import (
	"common/rabbitmq"
	"encoding/json"
	"fmt"
	"log"
	"user/models"
	"user/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func StartUserEventsConsumer(db *gorm.DB) {
	repo := repository.NewProfileRepository(db)

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
			ID:       userID,
			Username: name,
			FullName: name,
		}

		_, err := repo.FindByID(userID)
		if err == nil {
			fmt.Printf("Ошибка. Такой пользователь уже существует %s: %v", userID, err)
			return
		}

		if err = repo.Create(&profile); err != nil {
			log.Printf("Ошибка. Не удалось создать профиль %s: %v", userID, err)
			return
		}
	})
	if err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}
	log.Println("User events consumer started")
}
