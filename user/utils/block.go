package utils

import (
	"common/redis"
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

func PublishBlockEvent(blockerID, blockedID uuid.UUID, isBlocked bool) error {
	ctx := context.Background()
	payload := map[string]interface{}{
		"type":       "block_update",
		"blocker_id": blockerID.String(),
		"blocked_id": blockedID.String(),
		"is_blocked": isBlocked,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Println("blocked", isBlocked, blockedID.String())
	return redis.EventsRedisClient().Publish(ctx, "block:events", bytes).Err()
}
