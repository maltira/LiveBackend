package utils

import (
	"context"
	"encoding/json"
	"user/pkg/redis"

	"github.com/google/uuid"
)

const (
	BlockEventType  = "block_update"
	StatusEventType = "status_update"
)

type BlockEvent struct {
	EventType string `json:"event_type"`
	BlockerID string `json:"blocker_id"`
	BlockedID string `json:"blocked_id"`
	IsBlocked bool   `json:"is_blocked"`
}
type StatusEvent struct {
	EventType string `json:"event_type"`
	UserID    string `json:"user_id"`
	IsOnline  bool   `json:"is_online"`
	LastSeen  string `json:"last_seen"`
}

func PublishBlockEvent(blockerID, blockedID uuid.UUID, isBlocked bool) error {
	ctx := context.Background()
	payload := BlockEvent{
		EventType: BlockEventType,
		BlockerID: blockerID.String(),
		BlockedID: blockedID.String(),
		IsBlocked: isBlocked,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return redis.UserRedis.Publish(ctx, "user:block:events", bytes).Err()
}

func PublishStatusEvent(userID uuid.UUID, online bool, lastSeen string) error {
	ctx := context.Background()
	event := StatusEvent{
		EventType: StatusEventType,
		UserID:    userID.String(),
		IsOnline:  online,
		LastSeen:  lastSeen,
	}
	// log.Printf("Опубликовано событие %s (online: %t, last_seen: %s)", userID, online, lastSeen)
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return redis.UserRedis.Publish(ctx, "user:status:events", bytes).Err()
}
