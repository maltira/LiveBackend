package handler

import (
	"auth/config"
	"auth/internal/dto"
	"auth/internal/service"
	"auth/pkg/rabbitmq"
	"auth/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	sc service.AuthService
}

func NewAuthHandler(sc service.AuthService) *AuthHandler {
	return &AuthHandler{sc: sc}
}

// Register
// @Summary Регистрация нового пользователя
// @Description Создаёт нового пользователя с указанным email и паролем. После успешной регистрации отправляется OTP-код
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.AuthRequest true "Данные для регистрации"
// @Success 200  {object} dto.OTPSentResponse "Подтвердите регистрацию"
// @Failure      400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure      409  {object} dto.ErrorResponse "Пользователь с таким email уже существует"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var input dto.AuthRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the body"})
		return
	}

	// Создаём пользователя
	id, err := h.sc.Register(input.Email, input.Password)
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, dto.ErrorResponse{Code: 409, Error: err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	// Отправляем OTP
	_, _, err = h.sc.SendOTP(id, input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate OTP"})
		return
	}

	c.JSON(http.StatusOK, dto.OTPSentResponse{
		UserID:  id,
		Message: "OTP-код отправлен на указанную почту",
	})
}

// Login
// @Summary      Вход в систему
// @Description  Аутентифицирует пользователя по email и паролю. При успехе отправляется OTP-код.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.AuthRequest true "Данные для входа"
// @Success      200  {object} dto.OTPSentResponse "Подтвердите вход"
// @Failure      400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure      401  {object} dto.ErrorResponse "Неверный email или пароль"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input dto.AuthRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "incorrect data was transmitted in the body"})
		return
	}

	id, err := h.sc.Login(input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid username or password"})
		return
	}

	// Отправляем OTP
	_, _, err = h.sc.SendOTP(id, input.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate OTP"})
		return
	}

	c.JSON(http.StatusOK, dto.OTPSentResponse{
		UserID:  id,
		Message: "OTP-код отправлен на указанную почту",
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
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the body"})
		return
	}

	otp, err := h.sc.FindValidOTP(req.UserID, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid or expired OTP"})
		return
	}

	// Помечаем как использованный
	err = h.sc.MarkOTPAsUsed(otp.ID)

	user, err := h.sc.GetUserByID(req.UserID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "invalid user"})
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	device := utils.ParseDeviceInfo(userAgent)

	if req.Action == "login" {
		if user.ToBeDeletedAt != nil {
			recoveryToken, err := utils.GenerateTempToken(user.ID, 15*time.Minute, "recovery_token")
			if err != nil {
				c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate recovery token"})
				return
			}
			c.JSON(http.StatusOK, dto.RecoveryResponse{
				Message:       "Восстановление аккаунта",
				RecoveryToken: recoveryToken,
			})
			return
		}
		access, refresh, _ := h.sc.GenerateTokens(req.UserID, ip, userAgent, device)
		c.SetCookie("access_token", access, 15*60, "/", "", false, true)
		c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)

		c.JSON(http.StatusOK, dto.MessageResponse{
			Message: "Успешная авторизация",
		})
		return
	} else if req.Action == "register" {
		user.IsVerified = true
		err = h.sc.UpdateUser(user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "account verification error, please try again to register"})
			return
		}

		access, refresh, _ := h.sc.GenerateTokens(req.UserID, ip, userAgent, device)
		c.SetCookie("access_token", access, 15*60, "/", "", false, true)
		c.SetCookie("refresh_token", refresh, 30*24*60*60, "/", "", false, true)

		// сообщаем, что пользователь создан
		payload := map[string]interface{}{
			"user_id": user.ID,
			"action":  "user_created",
		}
		payloadBytes, _ := json.Marshal(payload)
		err = rabbitmq.Publish("user.events", payloadBytes)
		if err != nil {
			fmt.Printf("Failed to publish user_verified event: %v", err)
		}

		c.JSON(http.StatusOK, dto.MessageResponse{
			Message: "Успешная регистрация",
		})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Почта подтверждена",
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
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")

	if err != nil || refreshToken == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "missing refresh token"})
		return
	}

	access, refresh, err := h.sc.Refresh(refreshToken)
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

func (h *AuthHandler) ResendOTP(c *gin.Context) {
	id := c.DefaultQuery("id", "")
	email := c.DefaultQuery("email", "")

	if id == "" || email == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Требуются ID и Email"})
		return
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Некорректный UUID"})
		return
	}

	_, _, err = h.sc.SendOTP(userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка генерации OTP-кода"})
		return
	}
	c.JSON(http.StatusOK, dto.MessageResponse{Message: "OTP-код сгенерирован"})
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
func (h *AuthHandler) LogoutCurrent(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Missing refresh token"})
		return
	}

	// Отзываем текущий refresh-токен
	if err := h.sc.RevokeRefreshToken(refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// Добавляем текущий access-токен в blacklist
	accessToken, _ := c.Cookie("access_token")
	if accessToken != "" {
		err = h.sc.BlacklistAccessToken(c, accessToken)
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
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	// Отзываем все refresh-токены
	if err := h.sc.RevokeAllRefreshTokens(userID); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Logout all failed"})
		return
	}

	// Добавляем текущий access-токен в blacklist
	accessToken, _ := c.Cookie("access_token")
	if accessToken != "" {
		_ = h.sc.BlacklistAccessToken(c, accessToken)
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
// @Success      200  {object} dto.OTPSentResponse "Подтвердите сброс пароля"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	email := c.DefaultQuery("email", "")
	if email == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the param"})
		return
	}

	user, err := h.sc.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, dto.ErrorResponse{Code: 404, Error: "Аккаунта с такими данными не существует"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}
	if user.ToBeDeletedAt != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Аккаунт находится на стадии удаления, невозможно изменить пароль"})
		return
	}

	// Отправляем OTP
	_, _, err = h.sc.SendOTP(user.ID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate OTP"})
		return
	}

	c.JSON(http.StatusOK, dto.OTPSentResponse{
		UserID:  user.ID,
		Message: "OTP-код отправлен на указанную почту",
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
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Incorrect data was transmitted in the body"})
		return
	}

	// Меняем пароль
	err := h.sc.UpdatePassword(req.UserID, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// Выходим со всех устройств
	_ = h.sc.RevokeAllRefreshTokens(req.UserID)
	accessToken, _ := c.Cookie("access_token")
	if accessToken != "" {
		_ = h.sc.BlacklistAccessToken(c, accessToken)
	}
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Password reset successful"})
}

// ! Информация для пользователя

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
func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.MustGet("userID").(uuid.UUID)

	user, err := h.sc.GetUserByID(userID)
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

// ListSessions
// @Summary      Получить список активных сессий
// @Description  Возвращает список всех устройств/браузеров, с которых пользователь сейчас залогинен (активные refresh-токены)
// @Tags         user
// @Produce      json
// @Success      200  {array} dto.SessionResponse "Список сессий"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/sessions [get]
func (h *AuthHandler) ListSessions(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	sessions, err := h.sc.ListActiveSessions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	var response []dto.SessionResponse
	for _, s := range sessions {
		response = append(response, dto.SessionResponse{
			ID:        s.ID,
			Device:    s.Device,
			IP:        s.IP,
			UserAgent: s.UserAgent,
			CreatedAt: s.ExpiresAt.Add(-config.Env.RefreshTokenDuration),
			ExpiresAt: s.ExpiresAt,
		})
	}
	c.JSON(http.StatusOK, response)
}

// ! Удаление аккаунта

// Delete
// @Summary      Запрос на удаление аккаунта
// @Description  Повторный ввод пароля, генерация токена удаления
// @Tags         user
// @Produce      json
// @Success      200  {object} dto.TempTokenResponse "Токен для удаления"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/delete [post]
func (h *AuthHandler) Delete(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	var req dto.DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "В теле запроса переданы некорректные данные"})
		return
	}

	// Подтверждение пароля
	user, err := h.sc.GetUserByID(userID)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Invalid password"})
		return
	}

	// Генерируем temp-токен для подтверждения удаления
	deleteToken, err := utils.GenerateTempToken(user.ID, 15*time.Minute, "delete_token")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "failed to generate delete token"})
		return
	}
	c.JSON(http.StatusOK, dto.TempTokenResponse{
		UserID:    userID,
		TempToken: deleteToken,
	})
}

// DeleteConfirm
// @Summary      Подтверждение удаления аккаунта
// @Description  Отложенно удаляет аккаунт на 3 дня
// @Tags         user
// @Produce      json
// @Success      200  {object} dto.MessageResponse "Успешный запрос"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      401  {object} dto.ErrorResponse "Некорректный токен"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/delete/confirm [post]
func (h *AuthHandler) DeleteConfirm(c *gin.Context) {
	deleteToken := c.DefaultQuery("token", "")
	if deleteToken == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Invalid token"})
		return
	}
	// Проверяем временный токен
	claims, err := utils.ValidateTempToken(deleteToken)
	if err != nil || claims["action"] != "delete_token" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Invalid or expired delete token"})
		return
	}

	userID := claims["id"].(string)
	userUUID := uuid.MustParse(userID)

	deletionTime := time.Now().Add(3 * 24 * time.Hour)
	err = h.sc.ScheduleDeletion(userUUID, deletionTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Failed to schedule deletion"})
		return
	}

	// Выходим со всех устройств
	_ = h.sc.RevokeAllRefreshTokens(userUUID)
	accessToken, _ := c.Cookie("access_token")
	if accessToken != "" {
		_ = h.sc.BlacklistAccessToken(c, accessToken)
	}
	c.SetCookie("access_token", "", -1, "/", "", false, true)
	c.SetCookie("refresh_token", "", -1, "/", "", false, true)

	c.JSON(http.StatusOK, dto.MessageResponse{
		Message: "Аккаунт будет удалён через 3 дня. Вы можете восстановить доступ в любое время до истечения этого срока.",
	})
}

// DeleteCancel
// @Summary      Восстановление аккаунта (после удаления)
// @Description  Восстанавливает удаленный аккаунт
// @Tags         user
// @Produce      json
// @Success      200  {object} dto.MessageResponse "Аккаунт восстановлен"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      401  {object} dto.ErrorResponse "Некорректный токен"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/delete/cancel [post]
func (h *AuthHandler) DeleteCancel(c *gin.Context) {
	recoveryToken := c.DefaultQuery("token", "")
	if recoveryToken == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: "Invalid token"})
		return
	}
	// Проверяем временный токен
	claims, err := utils.ValidateTempToken(recoveryToken)
	if err != nil || claims["action"] != "recovery_token" {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{Code: 401, Error: "Invalid or expired recovery token"})
		return
	}
	userID := claims["id"].(string)
	userUUID := uuid.MustParse(userID)

	err = h.sc.CancelDeletion(userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.MessageResponse{Message: "Аккаунт успешно восстановлен"})
}
