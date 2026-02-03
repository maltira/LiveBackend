package handler

import (
	"net/http"
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
// @Success      200  {array} models.Settings "Настройки пользователя"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/settings [get]
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	id := c.MustGet("id").(uuid.UUID)

	settings, err := h.sc.GetSettings(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// SaveSettings
// @Summary      Обновить настройки
// @Description  Сохраняет обновленные настройки текущего пользователя
// @Tags         settings
// @Accept       json
// @Produce      json
// @Param        body body dto.SettingsUpdateRequest true "Обновленные настройки"
// @Success      200  {object} dto.MessageResponse "Настройки сохранены"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные или нечего обновлять"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/settings [put]
func (h *SettingsHandler) SaveSettings(c *gin.Context) {
	id := c.MustGet("id").(uuid.UUID)

	var req dto.SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректные данные в теле запроса"})
		return
	}

	err := h.sc.SaveSettings(id, &req)
	if err != nil {
		if err.Error() == "no settings to update" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Настройки сохранены"})
}
