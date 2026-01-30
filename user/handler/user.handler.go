package handler

import (
	"common/redis"
	"context"
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

// GetProfile
// @Summary      Получить профиль текущего пользователя
// @Description  Возвращает полный профиль авторизованного пользователя (username, full_name, bio, avatar_url и т.д.)
// @Tags         profile
// @Produce      json
// @Success      200  {object} models.Profile "Профиль пользователя"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      404  {object} dto.ErrorResponse "Профиль не найден"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile [get]
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

// FindProfile
// @Summary      Получить профиль пользователя по ID
// @Description  Возвращает публичные данные профиля другого пользователя
// @Tags         profile
// @Produce      json
// @Param        id   path   string  true   "ID пользователя (UUID)" Format(uuid)
// @Success      200  {object} models.Profile "Публичный профиль"
// @Failure      400  {object} dto.ErrorResponse "Некорректный ID"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      404  {object} dto.ErrorResponse "Пользователь не найден"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile/{id} [get]
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

// FindAll
// @Summary      Получить список всех профилей
// @Description  Возвращает список всех пользователей
// @Tags         profile
// @Produce      json
// @Success      200  {array} models.Profile "Список профилей"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile/all [get]
func (h *ProfileHandler) FindAll(c *gin.Context) {
	profiles, err := h.sc.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

// UpdateProfile
// @Summary      Обновить профиль текущего пользователя
// @Description  Позволяет изменить username, full_name, bio и другие публичные поля
// @Tags         profile
// @Accept       json
// @Produce      json
// @Param        body body dto.UpdateProfileRequest true "Новые данные профиля"
// @Success      200  {object} dto.MessageResponse "Профиль успешно обновлён"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные или username занят"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile [put]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	id := c.MustGet("userID").(uuid.UUID)

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректные данные в теле запроса"})
		return
	}
	if err := h.sc.Update(id, &req); err != nil {
		if err.Error() == "это имя пользователя занято" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "это имя пользователя занято"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Профиль успешно обновлен"})
}

// IsUsernameFree
// @Summary      Проверка доступности username
// @Description  Проверяет, свободен ли указанный username для использования
// @Tags         profile
// @Produce      json
// @Param        username path string true "Желаемый username"
// @Success      200  {object} bool "true — свободен, false — занят"
// @Failure      400  {object} dto.ErrorResponse "Некорректный username"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/profile/username/{username}/check [get]
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

func (h *ProfileHandler) GetUserStatus(c *gin.Context) {
	profileID := c.Param("id")
	profileUUID := uuid.MustParse(profileID)

	profile, err := h.sc.FindByID(profileUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	online, err := redis.OnlineRedisClient().Exists(context.Background(), "user:online:"+profileID).Result()
	if online > 0 {
		if !profile.Settings.ShowOnlineStatus {
			c.JSON(http.StatusOK, dto.ProfileStatusResponse{Online: false})
			return
		}
		c.JSON(http.StatusOK, dto.ProfileStatusResponse{Online: true})
		return
	}

	c.JSON(http.StatusOK, dto.ProfileStatusResponse{Online: false, LastSeen: *profile.LastSeen})
}
