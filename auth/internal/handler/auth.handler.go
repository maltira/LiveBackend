package handler

import (
	"auth/config"
	"auth/internal/dto"
	"auth/internal/service"
	"auth/pkg/rabbitmq"
	"auth/pkg/utils"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthHandler struct {
	asc service.AuthService
	osc service.OtpService
	tsc service.TokenService
}

func NewAuthHandler(asc service.AuthService, osc service.OtpService, tsc service.TokenService) *AuthHandler {
	return &AuthHandler{asc: asc, osc: osc, tsc: tsc}
}

// Register
// @Summary Регистрация нового пользователя
// @Description Создаёт нового пользователя с указанным email и паролем
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.AuthRequest true "Данные для регистрации"
// @Success 	200  {boolean} true "Подтвердите аккаунт по ссылке на почте"
// @Failure     400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure     409  {object} dto.ErrorResponse "Пользователь с таким email уже существует"
// @Failure     500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router      /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	err := h.asc.Register(req.Email, req.Password)
	if err != nil {
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, dto.ErrorResponse{Code: 409, Error: "Пользователь с такой почтой уже существует"})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка регистрации: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, true)
}

// VerifyEmail
// @Summary Верификация нового аккаунта
// @Description Активирует новый аккаунт, после верификации нужно выполнить вход повторно
// @Tags auth
// @Accept json
// @Produce json
// @Param		id path string true "Токен верификации"
// @Success 	200  {boolean} true
// @Failure     400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure     500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router      /auth/verify-email/{token} [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	user, err := h.asc.VerifyNewAccount(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 500, Error: err.Error()})
		return
	}

	// событие в очередь
	payload := map[string]interface{}{"user_id": user.ID, "action": "user_created"}
	if data, err := json.Marshal(payload); err == nil {
		_ = rabbitmq.Publish("user.events", data)
	}

	c.JSON(http.StatusOK, true)
}

// Login
// @Summary      Вход в систему
// @Description  Аутентифицирует пользователя по email и паролю. При успехе отправляется OTP-код.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.AuthRequest true "Данные для входа"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure      403  {object} dto.ErrorResponse "Неверный email или пароль"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.AuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	id, err := h.asc.Login(req.Email, req.Password)
	if err != nil {
		switch {
		case err.Error() == "account is not verified":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{Code: 403, Error: config.NotVerifiedError})
		case err.Error() == "invalid credentials":
			c.JSON(http.StatusForbidden, dto.ErrorResponse{Code: 403, Error: config.IncorrectAuthError})
		default:
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	err = h.osc.SendOTP(id, req.Email)
	if err != nil {
		log.Println("Ошибка генерации OTP:" + err.Error())
		// не прерываемся, тк пользователь сможет переотправить код
	}

	// отправляем на верификацию кода
	c.JSON(http.StatusOK, true)
}

// ! Выход из профиля

// LogoutCurrent
// @Summary      Выход из системы
// @Description  Завершает текущую сессию пользователя, отзывает токены и очищает cookie.
// @Tags         logout
// @Produce      json
// @Success      200  {boolean} true
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/logout [post]
func (h *AuthHandler) LogoutCurrent(c *gin.Context) {
	refreshToken, _ := c.Cookie("refresh_token")
	if err := h.tsc.RevokeRefreshToken(refreshToken); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка отзыва токена: " + err.Error()})
		return
	}
	utils.ClearAuthCookies(c)

	c.JSON(http.StatusOK, true)
}

// LogoutAll
// @Summary      Выход из системы
// @Description  Завершает все сессии пользователя, кроме текущей
// @Tags         logout
// @Produce      json
// @Success      200  {boolean} true
// @Failure      403  {object} dto.ErrorResponse "Некорректный токен"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/logout/all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	refreshToken, _ := c.Cookie("refresh_token")

	if err := h.tsc.RevokeAllRefreshTokens(userID, &refreshToken); err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка отзыва токенов: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, true)
}

// ! Сброс пароля

// ForgotPassword
// @Summary      Запрос на восстановление пароля
// @Description  Отправляет OTP-код для сброса пароля
// @Tags         reset
// @Accept       json
// @Produce      json
// @Param 		 email query string true "Email"
// @Success      200  {object} dto.OTPSentResponse "Подтвердите сброс пароля"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      403  {object} dto.ErrorResponse "Невозможно удалить аккаунт"
// @Failure      404  {object} dto.ErrorResponse "Аккаунт не найден"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	email := c.DefaultQuery("email", "")
	if email == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	user, err := h.asc.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка получения пользователя: " + err.Error()})
		return
	}
	if user.ToBeDeletedAt != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 403, Error: "Аккаунт находится на стадии удаления, невозможно изменить пароль"})
		return
	}

	// Отправляем OTP
	err = h.osc.SendOTP(user.ID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка генерации OTP: " + err.Error()})
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
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	err := h.asc.UpdatePassword(req.UserID, req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка обновления: " + err.Error()})
		return
	}

	_ = h.tsc.RevokeAllRefreshTokens(req.UserID, nil)

	utils.ClearAuthCookies(c)

	c.JSON(http.StatusOK, true)
}

// ! Информация для пользователя

// Me
// @Summary      Получить информацию о текущем пользователе
// @Description  Возвращает данные авторизованного пользователя
// @Tags         auth
// @Produce      json
// @Success      200  {object} dto.User "Информация о пользователе"
// @Failure      404  {object} dto.ErrorResponse "Пользователь не найден"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	user, err := h.asc.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}

// ChangeMail
// @Summary Смена почты
// @Description Позволяет изменить старую почту на новую
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ChangeEmailRequest true "Данные новой почты"
// @Success 	200  {boolean} true
// @Failure     400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure     404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure     500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router      /auth/change-mail [post]
func (h *AuthHandler) ChangeMail(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)

	var req dto.ChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	user, err := h.asc.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	err = h.osc.SendOTP(user.ID, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка генерации OTP: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, true)
}

// ChangePass
// @Summary Смена пароля
// @Description Позволяет изменить старый пароль на новый
// @Tags auth
// @Accept json
// @Produce json
// @Param body body dto.ChangeEmailRequest true "Старый и новый пароль"
// @Success 	200  {boolean} true
// @Failure     400  {object} dto.ErrorResponse "Некорректные входные данные"
// @Failure     403  {object} dto.ErrorResponse "Неверный данные, доступ запрещен"
// @Failure     404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure     500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router      /auth/change-pass [post]
func (h *AuthHandler) ChangePass(c *gin.Context) {
	userID, _ := c.MustGet("userID").(uuid.UUID)

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	user, err := h.asc.GetUserByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: err.Error()})
		}
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		c.JSON(403, dto.ErrorResponse{Code: 403, Error: "Указан неверный пароль от аккаунта"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.NewPassword)) == nil {
		c.JSON(403, dto.ErrorResponse{Code: 403, Error: "Новый пароль должен отличаться от текущего"})
		return
	}

	err = h.osc.SendOTP(user.ID, user.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка генерации OTP: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, true)
}

// ! Удаление аккаунта

// Delete
// @Summary      Запрос на удаление аккаунта
// @Description  Повторный ввод пароля, генерация токена удаления
// @Tags         delete
// @Produce      json
// @Success      200  {object} dto.OTPSentResponse "Подтвердите удаление"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/delete [post]
func (h *AuthHandler) Delete(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	email := c.Param("email")

	err := h.osc.SendOTP(userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка генерации OTP: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.OTPSentResponse{
		UserID:  userID,
		Message: "OTP-код отправлен на указанную почту",
	})
}

// RecoveryAccount
// @Summary      Восстановление аккаунта (после удаления)
// @Description  Восстанавливает удаленный аккаунт
// @Tags         delete
// @Produce      json
// @Param		 id path string true "uuid пользователя"
// @Param		 token query string true "recovery_token"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      403  {object} dto.ErrorResponse "Некорректный токен"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/recovery/{id} [put]
func (h *AuthHandler) RecoveryAccount(c *gin.Context) {
	id := c.Param("id")
	userID := uuid.MustParse(id)
	recoveryToken := c.DefaultQuery("token", "")
	if recoveryToken == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": recovery_token is empty"})
		return
	}

	claims, err := utils.ValidateTempToken(recoveryToken)
	if err != nil || claims["action"] != "recovery_token" {
		c.JSON(http.StatusForbidden, dto.ErrorResponse{Code: 403, Error: config.InvalidTokenError})
		return
	}

	err = h.asc.CancelDeletion(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка восстановления: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, true)
}
