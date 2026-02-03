package redis

import (
	"auth/config"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var AuthRedis *redis.Client

func InitAuthRedis() *redis.Client {
	AuthRedis = redis.NewClient(&redis.Options{
		Addr:     config.Env.RedisAddr,
		Password: "",
		DB:       0,
	})

	fmt.Println("REDIS ADDR:", config.Env.RedisAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := AuthRedis.Ping(ctx).Err(); err != nil {
		panic("не удалось подключиться к AuthRedis: " + err.Error())
	}
	return AuthRedis
}

func Close() {
	if AuthRedis != nil {
		if err := AuthRedis.Close(); err != nil {
			log.Printf("AuthRedis closing error: %v", err)
		}
		AuthRedis = nil
	}
	log.Println("AuthRedis connection closed")
}
