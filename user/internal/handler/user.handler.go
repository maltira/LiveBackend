package handler

import (
	"errors"
	"net/http"
	"strconv"
	"user/config"
	"user/internal/models/dto"
	"user/internal/service"
	"user/pkg/websocket"

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

// CreateProfile
// @Summary      Создать профиль
// @Description  Создаёт новый профиль пользователя
// @Tags         profile
// @Produce      json
// @Success      200  {object} dto.MessageResponse "Профиль создан"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile [post]
func (h *ProfileHandler) CreateProfile(c *gin.Context) {
	var req dto.CreateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": " + err.Error()})
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": " + err.Error()})
		return
	}

	if err = h.sc.Create(userID); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.MessageResponse{Success: true, Message: "Профиль успешно создан"})
}

// GetCurrentProfile
// @Summary      Получить профиль текущего пользователя
// @Description  Возвращает полный профиль авторизованного пользователя (username, full_name, bio, avatar_url и т.д.)
// @Tags         profile
// @Produce      json
// @Security	 BearerAuth
// @Success      200  {object} models.Profile "Профиль пользователя"
// @Failure      404  {object} dto.ErrorResponse "Профиль не найден"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile [get]
func (h *ProfileHandler) GetCurrentProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}

	profile, err := h.sc.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, profile)
}

// GetProfileByID
// @Summary      Получить профиль пользователя по ID
// @Description  Возвращает публичные данные профиля другого пользователя
// @Tags         profile
// @Produce      json
// @Security	 BearerAuth
// @Param        id   path   string  true   "ID пользователя (UUID)" Format(uuid)
// @Success      200  {object} models.Profile "Публичный профиль"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      404  {object} dto.ErrorResponse "Пользователь не найден"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile/{id} [get]
func (h *ProfileHandler) GetProfileByID(c *gin.Context) {
	id := c.Param("id")

	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}

	user, err := h.sc.FindByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
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
// @Security	 BearerAuth
// @Success      200  {array} models.Profile "Список профилей"
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

// GetProfilesByQuery
// @Summary      Список профилей по запросу
// @Description  Возвращает список пользователей, удовлетворящих запросу
// @Tags         profile
// @Produce      json
// @Security	 BearerAuth
// @Success      200  {array} models.Profile "Список профилей"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile/search [get]
func (h *ProfileHandler) GetProfilesByQuery(c *gin.Context) {
	query := c.Query("q")              // сам запрос
	l := c.DefaultQuery("limit", "10") // лимит кол-ва профилей

	if query == "" {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": query is empty"})
		return
	}

	limit, err := strconv.Atoi(l)
	if err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": limit"})
		return
	} else if limit < 1 || limit > 20 {
		limit = 5
	}

	profiles, err := h.sc.GetAllBySearch(query, limit)
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
// @Security	 BearerAuth
// @Param        body body dto.UpdateProfileRequest true "Новые данные профиля"
// @Success      200  {object} dto.MessageResponse "Данные изменены"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /user/profile [put]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("X-User-ID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}
	var req map[string]interface{}
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректные данные в теле запроса"})
		return
	}

	if err = h.sc.Update(userID, &req); err != nil {
		errorText := ""
		switch err.Error() {
		case "username error":
			errorText = "Имя пользователя должно содержать от 4 до 16 символов"
		case "username exists":
			errorText = "Имя пользователя уже занято"
		case "full_name error":
			errorText = "Имя должно содержать от 1 до 100 символов"
		case "bio error":
			errorText = "Описание должно содержить не более 500 символов"
		case "no columns to update":
			errorText = "Не передано ни одного поля"
		}
		if errorText != "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: errorText})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Success: true, Message: "Новые данные успешно сохранены"})
}

// IsUsernameFree
// @Summary      Проверка доступности username
// @Description  Проверяет, свободен ли указанный username для использования
// @Tags         profile
// @Produce      json
// @Security	 BearerAuth
// @Param        u query string true "Username"
// @Success      200  {boolean} bool "true — свободен, false — занят"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/profile/check-username [get]
func (h *ProfileHandler) IsUsernameFree(c *gin.Context) {
	username := c.Query("u")
	if len(username) < 4 {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Имя пользователя должно содержать от 4 до 16 символов"})
		return
	}

	isFree, err := h.sc.IsUsernameFree(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, isFree)
}

// GetUserStatus
// @Summary      Проверка статуса online
// @Description  Проверяет, в сети ли сейчас пользователь
// @Tags         profile
// @Produce      json
// @Security	 BearerAuth
// @Param        u path string true "Username"
// @Success      200  {object} dto.ProfileStatusResponse "Данные об онлайне"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /user/profile/check-status/{id} [get]
func (h *ProfileHandler) GetUserStatus(c *gin.Context) {
	id := c.Param("id")
	profileID := uuid.MustParse(id)

	profile, err := h.sc.FindByID(profileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	if !profile.Settings.ShowOnlineStatus {
		c.JSON(http.StatusOK, dto.ProfileStatusResponse{
			Online:   false,
			LastSeen: nil,
		})
		return
	}

	isOnline := websocket.IsClientOnline(profileID)
	response := dto.ProfileStatusResponse{
		Online:   isOnline,
		LastSeen: &profile.LastSeen,
	}

	c.JSON(http.StatusOK, response)
}
