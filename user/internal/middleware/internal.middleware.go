package middleware

import (
	"net/http"
	"os"
	"user/internal/models/dto"

	"github.com/gin-gonic/gin"
)

// InternalOnly проверяет заголовок X-Internal-Secret для inter-service вызовов.
// Секрет задаётся через переменную окружения INTERNAL_SECRET.
func InternalOnly() gin.HandlerFunc {
	secret := os.Getenv("INTERNAL_SECRET")
	return func(c *gin.Context) {
		if secret == "" || c.GetHeader("X-Internal-Secret") != secret {
			c.AbortWithStatusJSON(http.StatusForbidden, dto.ErrorResponse{Code: 403, Error: "forbidden"})
			return
		}
		c.Next()
	}
}
