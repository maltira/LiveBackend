package config

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	once      sync.Once
	authRedis *redis.Client
)

func initClients() {
	once.Do(func() {
		newClient := func(db int) *redis.Client {
			c := redis.NewClient(&redis.Options{
				Addr:     "localhost:6379",
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
	})
}

func AuthRedisClient() *redis.Client {
	initClients()
	return authRedis
}
