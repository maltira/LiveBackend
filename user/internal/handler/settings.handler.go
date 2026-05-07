package handler

import (
	"net/http"
	"user/config"
	"user/internal/models/dto"
	"user/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SettingsHandler struct {
	sc service.SettingsService
}

func NewSettingsHandler(sc service.SettingsService) *SettingsHandler {
	return &SettingsHandler{sc: sc}
}

// GetSettings
// @Summary      Получить настройки
// @Description  Возвращает настройки текущего пользователя
// @Tags         settings
// @Produce      json
// @Security	 BearerAuth
// @Success      200  {object} models.Settings "Настройки пользователя"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/settings [get]
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}

	settings, err := h.sc.GetSettings(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateVisibleStatus
// @Summary      Обновить отображение статуса
// @Description  Могут ли пользователи видеть мой статус в сети
// @Tags         settings
// @Accept       json
// @Produce      json
// @Security	 BearerAuth
// @Param        visible query string true "true или false"
// @Success      200  {object} dto.MessageResponse "Отображение статуса изменено"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/settings/status [put]
func (h *SettingsHandler) UpdateVisibleStatus(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}

	visible := c.Query("visible")
	if visible == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": visible is required"})
		return
	}

	err = h.sc.UpdateVisibleStatus(userID, visible == "true")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Success: true, Message: "Отображение статуса успешно изменено"})
}

// UpdateVisibleBirthDate
// @Summary      Обновить отображение даты рождения
// @Description  Кто может видеть мою дату рождения
// @Tags         settings
// @Accept       json
// @Produce      json
// @Security	 BearerAuth
// @Param        visible query string true "all или nobody"
// @Success      200  {object} dto.MessageResponse "Отображение даты изменено"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/settings/birth [put]
func (h *SettingsHandler) UpdateVisibleBirthDate(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}

	visible := c.Query("visible")
	if visible != "all" && visible != "nobody" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": visible is required"})
		return
	}

	err = h.sc.UpdateVisibleBirthDate(userID, visible)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Success: true, Message: "Отображение даты рождения успешно изменено"})
}
