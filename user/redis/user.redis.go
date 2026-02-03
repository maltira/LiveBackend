package redis

import (
	"common/config"
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var userRedis *redis.Client
var once sync.Once

func initUserRedis() {
	once.Do(func() {
		c := redis.NewClient(&redis.Options{
			Addr:     config.AppConfig.RedisAddr,
			Password: "",
			DB:       0,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.Ping(ctx).Err(); err != nil {
			panic("не удалось подключиться к UserRedis: " + err.Error())
		}
		userRedis = c
	})
}

func GetUserRedis() *redis.Client {
	initUserRedis()
	return userRedis
}

func Close() {
	if userRedis != nil {
		if err := userRedis.Close(); err != nil {
			log.Printf("UserRedis closing error: %v", err)
		}
		userRedis = nil
	}
	log.Println("UserRedis connection closed")
}
