package redis

import (
	"context"
	"gateway/config"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var GatewayRedis *redis.Client

func InitGatewayRedis() {
	GatewayRedis = redis.NewClient(&redis.Options{
		Addr:     config.Env.RedisAddr,
		Password: "",
		DB:       0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := GatewayRedis.Ping(ctx).Err(); err != nil {
		panic("не удалось подключиться к GatewayRedis: " + err.Error())
	}
}

func Close() {
	if GatewayRedis != nil {
		if err := GatewayRedis.Close(); err != nil {
			log.Printf("GatewayRedis closing error: %v", err)
		}
		GatewayRedis = nil
	}
	log.Println("GatewayRedis connection closed")
}
