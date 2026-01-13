package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimiterMiddleware(redisClient *redis.Client, rateString string, prefix string) gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted(rateString)
	if err != nil {
		panic("Invalid rate format: " + rateString + " → " + err.Error())
	}

	store, err := redisstore.NewStoreWithOptions(redisClient, limiter.StoreOptions{
		Prefix: prefix,
	})

	if err != nil {
		panic("Failed to create Redis store: " + err.Error())
	}

	lim := limiter.New(store, rate)

	mw := mgin.NewMiddleware(lim)
	return mw
}
