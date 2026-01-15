package main

import (
	"auth/dto"
	"auth/utils"
	"errors"
	"fmt"
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
// @Success 200  {object} dto.TempTokenResponse "Временный токен для подтверждения OTP"
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
	tempToken, err := utils.GenerateTempToken(id, 15*time.Minute, "register")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate temp token"})
		return
	}

	c.JSON(http.StatusOK, dto.TempTokenResponse{
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
// @Success      200  {object} dto.TempTokenResponse "Временный токен для ввода OTP"
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
	tempToken, err := utils.GenerateTempToken(id, 15*time.Minute, "login")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate temp token"})
		return
	}

	c.JSON(http.StatusOK, dto.TempTokenResponse{
		TempToken: tempToken,
	})
}

// VerifyOTP
// @Summary      Подтверждение OTP-кода
// @Description  Проверяет введённый пользователем OTP-код. При успехе выдаёт access и refresh токены в cookie.
// @Tags         otp
// @Accept       json
// @Produce      json
// @Param        body body dto.VerifyOTPRequest true "Данные для верификации"
// @Success      200  {object} dto.MessageResponse "Успешная аутентификация"
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
	action := claims["action"].(string)

	otp, err := h.service.repo.FindValidOTP(userUUID, input.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid or expired OTP"})
		return
	}

	// Помечаем как использованный
	err = h.service.repo.MarkOTPAsUsed(otp.ID)

	user, err := h.service.GetUserByID(userUUID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid user"})
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	device := utils.ParseDeviceInfo(userAgent)

	if action == "login" {
		access, refresh, _ := h.service.GenerateTokens(userUUID, ip, userAgent, device)
		c.SetCookie("access_token", access, 15*60, "/", "", false, true)
		c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)

		c.JSON(http.StatusOK, dto.MessageResponse{
			Message: "Login successful",
		})
		return
	} else if action == "register" {
		user.IsVerified = true
		err = h.service.UpdateUser(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "account verification error, please try again to register"})
			return
		}

		access, refresh, _ := h.service.GenerateTokens(userUUID, ip, userAgent, device)
		c.SetCookie("access_token", access, 15*60, "/", "", false, true)
		c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)

		c.JSON(http.StatusOK, dto.MessageResponse{
			Message: "Registration successful",
		})
		return
	} else if action == "forgot_password" {
		// Генерируем временный токен
		resetToken, err := utils.GenerateTempToken(user.ID, 15*time.Minute, "reset_token")
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate reset token"})
			return
		}
		fmt.Printf("\n*| reset_token: %s\n\n", resetToken) // как и OTP, надо отправлять на почту
		return
	}

	c.JSON(http.StatusOK, dto.TempTokenResponse{
		TempToken: input.TempToken,
	})
}

// Refresh
// @Summary      Обновление access-токена
// @Description  Обновляет access-токен с помощью refresh-токена из cookie. Выдаёт новые токены в cookie.
// @Tags         token
// @Produce      json
// @Success      200  {object} dto.MessageResponse "Токены успешно обновлены"
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
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: err.Error()})
		return
	}

	c.SetCookie("access_token", access, 15*60, "/", "", false, true)
	c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Refresh successful",
	})
}

// ! Выход из профиля

// LogoutCurrent
// @Summary      Выход из системы
// @Description  Завершает текущую сессию пользователя, отзывает токены и очищает cookie.
// @Tags         logout
// @Produce      json
// @Success      200  {object} dto.MessageResponse "Успешный выход"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/logout [post]
func (h *Handler) LogoutCurrent(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Missing refresh token"})
		return
	}

	// Отзываем текущий refresh-токен
	if err := h.service.RevokeRefreshToken(refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// Добавляем текущий access-токен в blacklist
	accessToken, _ := c.Cookie("access_token")
	if accessToken != "" {
		err = h.service.BlacklistAccessToken(c, accessToken)
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Logged out from current device successfully"})
}

// LogoutAll
// @Summary      Выход из системы
// @Description  Завершает все сессии пользователя, отзывает токены и очищает cookie.
// @Tags         logout
// @Produce      json
// @Success      200  {object} dto.MessageResponse "Успешный выход"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/logout/all [post]
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	// Отзываем все refresh-токены
	if err := h.service.RevokeAllRefreshTokens(userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Logout all failed"})
		return
	}

	// Добавляем текущий access-токен в blacklist
	accessToken, _ := c.Cookie("access_token")
	if accessToken != "" {
		_ = h.service.BlacklistAccessToken(c, accessToken)
	}

	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Logged out successfully"})
}

// ! Сброс пароля

// ForgotPassword
// @Summary      Запрос на восстановление пароля
// @Description  Отправляет OTP-код для сброса пароля (требует query param "?email=example@example").
// @Tags         reset
// @Accept       json
// @Produce      json
// @Success      200  {object} dto.TempTokenResponse "Временный токен для следующего шага"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/forgot-password [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	email := c.DefaultQuery("email", "")
	if email == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the param"})
		return
	}
	user, err := h.service.repo.FindByEmail(email)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, dto.ErrorResponse{Code: 401, Error: "OTP code was sent"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// Генерируем временный токен
	tempToken, err := utils.GenerateTempToken(user.ID, 15*time.Minute, "forgot_password")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate temp token"})
		return
	}

	// Отправляем OTP
	_, _, err = h.service.SendOTP(user.ID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate OTP"})
		return
	}

	c.JSON(http.StatusOK, dto.TempTokenResponse{
		TempToken: tempToken,
	})
}

// ResetPassword
// @Summary      Сброс пароля по OTP-токену
// @Description  Меняет пароль пользователя после успешной проверки OTP и временного токена.
// @Description  После успешного сброса автоматически выходит со всех устройств (отзывает все refresh-токены и добавляет текущий access в blacklist).
// @Tags         reset
// @Accept       json
// @Produce      json
// @Param        body body dto.ResetPasswordRequest true "Токен и новый пароль"
// @Success      200  {object} dto.MessageResponse "Пароль успешно изменён, выполнен выход со всех устройств"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      401  {object} dto.ErrorResponse "Недействительный или просроченный токен"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the body"})
		return
	}

	// Проверяем временный токен
	claims, err := utils.ValidateTempToken(req.ResetToken)
	if err != nil || claims["action"] != "reset_token" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Invalid or expired reset token"})
		return
	}

	// Меняем пароль
	userID := claims["id"].(string)
	userUUID := uuid.MustParse(userID)
	err = h.service.UpdatePassword(userUUID, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// Выходим со всех устройств
	_ = h.service.RevokeAllRefreshTokens(userUUID)
	accessToken, _ := c.Cookie("access_token")
	if accessToken != "" {
		_ = h.service.BlacklistAccessToken(c, accessToken)
	}
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Password reset successful"})
}

// ! Информация о пользователе

// Me
// @Summary      Получить информацию о текущем пользователе
// @Description  Возвращает данные авторизованного пользователя
// @Tags         user
// @Produce      json
// @Success      200  {object} dto.User "Информация о пользователе"
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
	fmt.Println(userID, user, err)
	c.JSON(http.StatusOK, user)
}
