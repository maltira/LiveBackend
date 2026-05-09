package middleware

import (
	"auth/internal/dto"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func ValidateUUID(field string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param(field)

		if _, err := uuid.Parse(id); err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, dto.ErrorResponse{Code: 400, Error: "invalid UUID"})
			return
		}
		c.Next()
	}
}
