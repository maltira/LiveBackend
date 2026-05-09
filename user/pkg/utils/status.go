package utils

import (
	"log"
	"time"

	"github.com/google/uuid"
)

// UpdateLastSeenFunc задаётся при старте приложения через InitStatusUtils
var UpdateLastSeenFunc func(userID uuid.UUID, lastSeen time.Time) error

func InitStatusUtils(fn func(userID uuid.UUID, lastSeen time.Time) error) {
	UpdateLastSeenFunc = fn
}

func SetOnline(userID uuid.UUID) {
	err := PublishStatusEvent(userID, true, time.Now())
	if err != nil {
		log.Printf("[SetOnline] Failed to publish online event for %s: %v", userID, err)
	}
}

func SetOffline(userID uuid.UUID) {
	t := time.Now()
	err := PublishStatusEvent(userID, false, t)
	if err != nil {
		log.Printf("[SetOffline] Failed to publish event for %s: %v", userID, err)
	}
	if UpdateLastSeenFunc != nil {
		if err = UpdateLastSeenFunc(userID, t); err != nil {
			log.Printf("[SetOffline] Failed to change last_seen for %s: %v", userID, err)
		}
	}
}
