package redis

import (
	"chat/config"
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var ChatRedis *redis.Client

func InitChatRedis() *redis.Client {
	ChatRedis = redis.NewClient(&redis.Options{
		Addr:     config.Env.RedisAddr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := ChatRedis.Ping(ctx).Err(); err != nil {
		panic("не удалось подключиться к UserRedis: " + err.Error())
	}
	return ChatRedis
}

func Close() {
	if ChatRedis != nil {
		if err := ChatRedis.Close(); err != nil {
			log.Printf("UserRedis closing error: %v", err)
		}
		ChatRedis = nil
	}
	log.Println("UserRedis connection closed")
}
