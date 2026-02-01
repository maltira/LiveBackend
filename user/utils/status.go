package utils

import (
	"common/redis"
	"context"
	"time"
	"user/repository"

	"github.com/google/uuid"
)

func SetOnline(userID uuid.UUID) error {
	key := "user:online:" + userID.String()
	return redis.EventsRedisClient().Set(context.Background(), key, "1", 60*time.Second).Err()
}

func SetOffline(userID uuid.UUID, r *repository.ProfileRepository) error {
	key := "user:online:" + userID.String()
	err := redis.EventsRedisClient().Del(context.Background(), key).Err()
	if err != nil {
		return err
	}
	return (*r).UpdateLastSeen(userID, time.Now())
}
