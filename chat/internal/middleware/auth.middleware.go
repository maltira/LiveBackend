package middleware

import (
	"chat/config"
	"chat/pkg/redis"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		accessToken, err := c.Cookie("access_token")
		if err != nil || accessToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		token, err := parseToken(accessToken)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		claims, _ := token.Claims.(jwt.MapClaims)

		jti, _ := claims["jti"].(string)
		if jti != "" {
			key := "auth:blacklist:access:" + jti
			exists, _ := redis.ChatRedis.Get(c.Request.Context(), key).Result()
			if exists != "" { // ключ существует -> токен отозван
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token revoked"})
				return
			}
		}

		c.Set("userID", uuid.MustParse(claims["id"].(string)))
		c.Next()
	}
}

func parseToken(tokenString string) (*jwt.Token, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return config.Env.JWTSecret, nil
	})
}
