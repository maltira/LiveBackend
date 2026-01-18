package handler

import (
	"errors"
	"net/http"
	"user/models/dto"
	"user/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserHandler struct {
	sc service.UserService
}

func NewUserHandler(sc service.UserService) *UserHandler {
	return &UserHandler{sc: sc}
}

func (h *UserHandler) FindUser(c *gin.Context) {
	id := c.Param("id")
	userID := uuid.MustParse(id)

	user, err := h.sc.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: "Пользователь не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}
