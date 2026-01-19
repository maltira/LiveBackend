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

type ProfileHandler struct {
	sc service.ProfileService
}

func NewProfileHandler(sc service.ProfileService) *ProfileHandler {
	return &ProfileHandler{sc: sc}
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	id := c.MustGet("userID").(uuid.UUID)

	profile, err := h.sc.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: "Пользователь не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *ProfileHandler) FindProfile(c *gin.Context) {
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

func (h *ProfileHandler) FindAll(c *gin.Context) {
	profiles, err := h.sc.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	id := c.MustGet("userID").(uuid.UUID)

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректные данные в теле запроса"})
		return
	}
	if err := h.sc.Update(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Профиль успешно обновлен"})
}

func (h *ProfileHandler) IsUsernameFree(c *gin.Context) {
	username := c.Param("username")

	if len(username) == 0 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "invalid username"})
		return
	}

	isFree, err := h.sc.IsUsernameFree(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, isFree)
}
