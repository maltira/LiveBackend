package redis

import (
	"common/config"
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var authRedis *redis.Client
var once sync.Once

func initAuthRedis() {
	once.Do(func() {
		c := redis.NewClient(&redis.Options{
			Addr:     config.AppConfig.RedisAddr,
			Password: "",
			DB:       0,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.Ping(ctx).Err(); err != nil {
			panic("не удалось подключиться к AuthRedis: " + err.Error())
		}
		authRedis = c
	})
}

func GetAuthRedis() *redis.Client {
	initAuthRedis()
	return authRedis
}

func Close() {
	if authRedis != nil {
		if err := authRedis.Close(); err != nil {
			log.Printf("AuthRedis closing error: %v", err)
		}
		authRedis = nil
	}
	log.Println("AuthRedis connection closed")
}
