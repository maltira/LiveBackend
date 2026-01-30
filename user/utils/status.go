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
	return redis.OnlineRedisClient().Set(context.Background(), key, "1", 60*time.Second).Err()
}

func SetOffline(userID uuid.UUID, r *repository.ProfileRepository) error {
	key := "user:online:" + userID.String()
	err := redis.OnlineRedisClient().Del(context.Background(), key).Err()
	if err != nil {
		return err
	}
	return (*r).UpdateLastSeen(userID, time.Now())
}
