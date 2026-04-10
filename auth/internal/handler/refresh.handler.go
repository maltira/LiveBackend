package handler

import (
	"auth/config"
	"auth/internal/dto"
	"auth/internal/service"
	"auth/pkg/utils"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshHandler struct {
	sc service.AuthService
}

func NewRefreshHandler(sc service.AuthService) *RefreshHandler {
	return &RefreshHandler{sc: sc}
}

// Refresh
// @Summary      Обновление access-токена
// @Description  Обновляет access-токен с помощью refresh-токена из cookie. Выдаёт новые токены в cookie.
// @Tags         token
// @Produce      json
// @Success      200  {boolean} true
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/refresh [post]
func (h *RefreshHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: config.UnauthorizedError})
		return
	}
	_, refresh, err := h.sc.Refresh(refreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": refresh_token"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка обновления токенов: " + err.Error()})
		return
	}

	utils.SetAuthCookies(c, refresh)

	c.JSON(http.StatusOK, true)
}

// TerminateSession
// @Summary      Закончить конкретную сессию
// @Description  Заканчивает указанную сессию
// @Tags         token
// @Produce      json
// @Success      200  {boolean} true
// @Param        token path string true "Refresh-токен сессии, которую нужно завершить"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/logout/{token} [delete]
func (h *RefreshHandler) TerminateSession(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": token"})
		return
	}

	err := h.sc.RevokeRefreshToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": refresh_token"})
			return
		}
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: "Ошибка удаления токена: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, true)
}

// ListSessions
// @Summary      Получить список активных сессий
// @Description  Возвращает список всех устройств/браузеров, с которых пользователь сейчас залогинен (активные refresh-токены)
// @Tags         token
// @Produce      json
// @Success      200  {array} dto.SessionResponse "Список сессий"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/sessions [get]
func (h *RefreshHandler) ListSessions(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	sessions, err := h.sc.ListActiveSessions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	var response []dto.SessionResponse
	for _, s := range sessions {
		response = append(response, dto.SessionResponse{
			ID:           s.ID,
			RefreshToken: s.Token,
			Device:       s.Device,
			IP:           s.IP,
			UserAgent:    s.UserAgent,
			CreatedAt:    s.ExpiresAt.Add(-config.Env.RefreshTokenDuration),
			ExpiresAt:    s.ExpiresAt,
		})
	}
	c.JSON(http.StatusOK, response)
}
