package utils

import (
	"log"
	"time"
	"user/internal/repository"

	"github.com/google/uuid"
)

func SetOnline(userID uuid.UUID) {
	err := PublishStatusEvent(userID, true, "")
	if err != nil {
		log.Printf("[SetOnline] Failed to publish online event for %s: %v", userID, err)
	}
}

func SetOffline(userID uuid.UUID, r *repository.ProfileRepository) {
	t := time.Now()
	err := PublishStatusEvent(userID, false, t.String())
	if err != nil {
		log.Printf("[SetOffline] Failed to publish event for %s: %v", userID, err)
	}
	err = (*r).UpdateLastSeen(userID, t)
	if err != nil {
		log.Printf("[SetOffline] Failed to change last_seen for %s: %v", userID, err)
	}
}
