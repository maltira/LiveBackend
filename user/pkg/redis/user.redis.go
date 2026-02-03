package redis

import (
	"context"
	"log"
	"time"
	"user/config"

	"github.com/redis/go-redis/v9"
)

var UserRedis *redis.Client

func InitUserRedis() *redis.Client {
	UserRedis = redis.NewClient(&redis.Options{
		Addr:     config.Env.RedisAddr,
		Password: "",
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := UserRedis.Ping(ctx).Err(); err != nil {
		panic("не удалось подключиться к UserRedis: " + err.Error())
	}
	return UserRedis
}

func Close() {
	if UserRedis != nil {
		if err := UserRedis.Close(); err != nil {
			log.Printf("UserRedis closing error: %v", err)
		}
		UserRedis = nil
	}
	log.Println("UserRedis connection closed")
}
