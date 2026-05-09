package middleware

import (
	"context"
	gwRedis "gateway/pkg/redis"
	"gateway/pkg/utils"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader { // не было префикса Bearer
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		parsedToken, err := utils.ParseToken(tokenString)
		if err != nil || !parsedToken.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// Проверяем, не отозван ли access-токен (blacklist по jti)
		if jti, exists := claims["jti"].(string); exists && jti != "" {
			blacklistKey := "blacklist:jti:" + jti
			val, err := gwRedis.GatewayRedis.Exists(context.Background(), blacklistKey).Result()
			if err != nil {
				log.Printf("[AuthMiddleware] Redis blacklist check error: %v", err)
				// При ошибке Redis пропускаем — не блокируем пользователя из-за сбоя Redis
			} else if val > 0 {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token has been revoked"})
				return
			}
		}

		refreshToken, err := c.Cookie("refresh_token")
		if err != nil || refreshToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired session"})
			return
		}

		c.Request.Header.Set("X-User-ID", claims["id"].(string))
		c.Next()
	}
}
