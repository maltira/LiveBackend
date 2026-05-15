package middleware

import "github.com/gin-gonic/gin"

// StripInternalHeaders удаляет заголовки, которые должны устанавливаться
// только gateway'ем (из JWT claims), а не клиентом.
func StripInternalHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Header.Del("X-User-ID")
		c.Request.Header.Del("X-Internal-Secret")
		c.Next()
	}
}
