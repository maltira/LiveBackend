package redis

import (
	"common/config"
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	once       sync.Once
	authRedis  *redis.Client
	eventRedis *redis.Client
)

func initClients() {
	once.Do(func() {
		newClient := func(db int) *redis.Client {
			c := redis.NewClient(&redis.Options{
				Addr:     config.AppConfig.RedisAddr,
				Password: "",
				DB:       db,
			})

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := c.Ping(ctx).Err(); err != nil {
				panic("не удалось подключиться к Redis (DB " + strconv.Itoa(db) + "): " + err.Error())
			}

			return c
		}

		authRedis = newClient(0)
		eventRedis = newClient(3)
	})
}

func AuthRedisClient() *redis.Client {
	initClients()
	return authRedis
}
func EventsRedisClient() *redis.Client {
	initClients()
	return eventRedis
}

func Close() {
	if authRedis != nil {
		if err := authRedis.Close(); err != nil {
			log.Printf("Ошибка закрытия Redis (auth): %v", err)
		}
		authRedis = nil
	}
	if eventRedis != nil {
		if err := eventRedis.Close(); err != nil {
			log.Printf("Ошибка закрыт Redis (event): %v", err)
		}
		eventRedis = nil
	}
	fmt.Println("redis connection closed")
}
