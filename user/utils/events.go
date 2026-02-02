package utils

import (
	"common/redis"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

const (
	BlockEventType  = "block_update"
	StatusEventType = "status_update"
)

type BlockEvent struct {
	Type      string `json:"type"`
	BlockerID string `json:"blocker_id"`
	BlockedID string `json:"blocked_id"`
	IsBlocked bool   `json:"is_blocked"`
}
type StatusEvent struct {
	Type     string `json:"type"`
	UserID   string `json:"user_id"`
	IsOnline bool   `json:"is_online"`
	LastSeen string `json:"last_seen"`
}

func PublishBlockEvent(blockerID, blockedID uuid.UUID, isBlocked bool) error {
	ctx := context.Background()
	payload := BlockEvent{
		Type:      BlockEventType,
		BlockerID: blockerID.String(),
		BlockedID: blockedID.String(),
		IsBlocked: isBlocked,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Println("blocked", isBlocked, blockedID.String())
	return redis.EventsRedisClient().Publish(ctx, "block:events", bytes).Err()
}

func PublishStatusEvent(userID uuid.UUID, online bool, lastSeen string) error {
	ctx := context.Background()
	event := StatusEvent{
		Type:     StatusEventType,
		UserID:   userID.String(),
		IsOnline: online,
		LastSeen: lastSeen,
	}
	// log.Printf("Опубликовано событие %s (online: %t, last_seen: %s)", userID, online, lastSeen)
	bytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return redis.EventsRedisClient().Publish(ctx, "status:events", bytes).Err()
}
