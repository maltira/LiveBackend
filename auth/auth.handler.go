package main

import (
	"auth/utils"
	"common/dto"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
}

func NewAuthHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register
// @Summary Регистрация нового пользователя
// @Description Создаёт нового пользователя с указанным email и паролем. После успешной регистрации отправляется OTP-код
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.AuthRequest true "Данные для регистрации"
// @Success 200  {object} dto.AuthResponse "Временный токен для подтверждения OTP"
// @Failure      400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure      409  {object} dto.ErrorResponse "Пользователь с таким email уже существует"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var input dto.AuthRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the body"})
		return
	}

	// Создаём пользователя
	id, err := h.service.Register(input.Email, input.Password)
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, dto.ErrorResponse{Code: 409, Error: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	// Отправляем OTP
	_, _, err = h.service.SendOTP(id, input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate OTP"})
		return
	}

	// Генерируем временный токен
	tempToken, err := utils.GenerateTempToken(id, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate temp token"})
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		Action:    "register",
		TempToken: tempToken,
	})
}

// Login
// @Summary      Вход в систему
// @Description  Аутентифицирует пользователя по email и паролю. При успехе отправляется OTP-код.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.AuthRequest true "Данные для входа"
// @Success      200  {object} dto.AuthResponse "Временный токен для ввода OTP"
// @Failure      400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure      401  {object} dto.ErrorResponse "Неверный email или пароль"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var input dto.AuthRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "incorrect data was transmitted in the body"})
		return
	}

	id, err := h.service.Login(input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid username or password"})
		return
	}

	// Отправляем OTP
	_, _, err = h.service.SendOTP(id, input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate OTP"})
		return
	}

	// Генерируем временный токен
	tempToken, err := utils.GenerateTempToken(id, 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate temp token"})
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		Action:    "login",
		TempToken: tempToken,
	})
}

// VerifyOTP
// @Summary      Подтверждение OTP-кода
// @Description  Проверяет введённый пользователем OTP-код. При успехе выдаёт access и refresh токены в cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.VerifyOTPRequest true "Данные для верификации"
// @Success      200  {object}dto.MessageResponse "Успешная аутентификация"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      401  {object} dto.ErrorResponse "Неверный/просроченный код или временный токен"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/verify [post]
func (h *Handler) VerifyOTP(c *gin.Context) {
	var input dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the body"})
		return
	}

	// Проверяем временный токен
	claims, err := utils.ValidateTempToken(input.TempToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Invalid or expired temp token"})
		return
	}
	userID := claims["id"].(string)
	userUUID, _ := uuid.Parse(userID)

	otp, err := h.service.repo.FindValidOTP(userUUID, input.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid or expired OTP"})
		return
	}

	// Помечаем как использованный
	err = h.service.repo.MarkOTPAsUsed(otp.ID)

	// Верифицируем пользователя
	if input.Action == "register" {
		user, err := h.service.GetUserByID(userUUID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid user"})
			return
		}
		user.IsVerified = true
		err = h.service.UpdateUser(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "account verification error, please try again to register"})
			return
		}
	}

	access, refresh, err := h.service.GenerateTokens(userUUID)
	c.SetCookie("access_token", access, 15*60, "/", "", false, true)
	c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Authentication successful",
	})
}

// Refresh
// @Summary      Обновление access-токена
// @Description  Обновляет access-токен с помощью refresh-токена из cookie. Выдаёт новые токены в cookie.
// @Tags         auth
// @Produce      json
// @Success      200  {object}dto.MessageResponse "Токены успешно обновлены"
// @Failure      401  {object} dto.ErrorResponse "Отсутствует или недействителен refresh-токен"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка"
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")

	if err != nil || refreshToken == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "missing refresh token"})
		return
	}

	access, refresh, err := h.service.Refresh(refreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid or expired refresh token"})
		return
	}

	c.SetCookie("access_token", access, 15*60, "/", "", false, true)
	c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Refresh successful",
	})
}

// Logout
// @Summary      Выход из системы
// @Description  Завершает сессию пользователя, отзывает токены и очищает cookie.
// @Tags         auth
// @Produce      json
// @Success      200  {object}dto.MessageResponse "Успешный выход"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	refreshToken, err := c.Cookie("refresh_token")

	if err != nil || refreshToken == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Missing refresh token"})
		return
	}

	if err := h.service.Logout(userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Logged out successfully"})
}

// Me
// @Summary      Получить информацию о текущем пользователе
// @Description  Возвращает данные авторизованного пользователя
// @Tags         auth
// @Produce      json
// @Success      200  {object} models.User "Информация о пользователе"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      404  {object} dto.ErrorResponse "Пользователь не найден"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	userID, _ := c.MustGet("userID").(uuid.UUID)

	user, err := h.service.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}
