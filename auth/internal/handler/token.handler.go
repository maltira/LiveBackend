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
	tsc service.TokenService
}

func NewRefreshHandler(tsc service.TokenService) *RefreshHandler {
	return &RefreshHandler{tsc: tsc}
}

// Refresh
// @Summary      Обновление токенов
// @Description  Обновляет access-токен и refresh-токен
// @Tags         token
// @Produce      json
// @Success      200  {object} dto.TokenResponse "Access токен"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/refresh [post]
func (h *RefreshHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: config.UnauthorizedError})
		return
	}

	access, refresh, err := h.tsc.Refresh(refreshToken)
	if err != nil {
		if err.Error() == "invalid refresh token" {
			utils.ClearAuthCookies(c)
			c.JSON(401, dto.ErrorResponse{Code: 401, Error: config.InvalidTokenError})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка обновления токенов: " + err.Error()})
		return
	}

	utils.SetAuthCookies(c, refresh)

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
	})
}

// TerminateSession
// @Summary      Закончить конкретную сессию
// @Description  Заканчивает указанную сессию
// @Tags         logout
// @Produce      json
// @Security     BearerAuth
// @Param        token path string true "Refresh-токен сессии, которую нужно завершить"
// @Success      200  {object} dto.MessageResponse "Сессия завершена"
// @Failure      400  {object} dto.ErrorResponse "Переданы некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/logout/{token} [post]
func (h *RefreshHandler) TerminateSession(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": token"})
		return
	}

	err := h.tsc.RevokeRefreshToken(token)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(500, dto.ErrorResponse{Code: 500, Error: "Ошибка удаления токена: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Success: true, Message: "Сессия завершена"})
}

// ListSessions
// @Summary      Получить список активных сессий
// @Description  Возвращает список всех устройств/браузеров, с которых пользователь сейчас залогинен (активные refresh-токены)
// @Tags         token
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array} models.RefreshToken "Список сессий"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/sessions [get]
func (h *RefreshHandler) ListSessions(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID активного пользователя"})
		return
	}

	sessions, err := h.tsc.ListActiveSessions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}
