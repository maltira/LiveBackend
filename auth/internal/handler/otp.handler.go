package handler

import (
	"auth/config"
	"auth/internal/dto"
	"auth/internal/models"
	"auth/internal/service"
	"auth/pkg/utils"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type OtpHandler struct {
	osc service.OtpService
	asc service.AuthService
	tsc service.TokenService
}

func NewOtpHandler(osc service.OtpService, asc service.AuthService, tsc service.TokenService) *OtpHandler {
	return &OtpHandler{osc: osc, asc: asc, tsc: tsc}
}

// VerifyOTP
// @Summary      Подтверждение OTP-кода
// @Description  Проверяет введённый пользователем OTP-код
// @Tags         otp
// @Accept       json
// @Produce      json
// @Param        body body dto.VerifyOTPRequest true "Данные для верификации"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      403  {object} dto.ErrorResponse "Доступ запрещён"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/verify [post]
func (h *OtpHandler) VerifyOTP(c *gin.Context) {
	var req dto.VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	otp, err := h.osc.FindValidOTP(req.UserID, req.Code)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.InvalidOtpError})
		return
	}

	if err = h.osc.MarkOTPAsUsed(otp.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": OTP-код"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка отметки OTP: " + err.Error()})
	}

	user, err := h.asc.FindByID(req.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
			return
		}
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка получения пользователя: " + err.Error()})
		return
	} else if user.ToBeDeletedAt != nil && time.Now().After(*user.ToBeDeletedAt) {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{Code: 404, Error: config.NotFoundError + ": Пользователь"})
		return
	}

	switch req.Action {
	case "login":
		h.handleLoginAction(c, user)
	case "change-mail":
		h.handleChangeMailAction(c, user, req.Email)
	case "change-pass":
		h.handleChangePasswordAction(c, user, req.Password)
	case "delete-account":
		h.handleDeleteAccountAction(c, user)
	default:
		c.JSON(404, dto.ErrorResponse{Code: 404, Error: "Указанное действие не найдено"})
	}
}

// ResendOTP
// @Summary      Повторить отправку OTP
// @Description  Позволяет заново сгенерировать и отправить OTP-код
// @Tags         otp
// @Accept       json
// @Produce      json
// @Param 		 id query string true "uuid пользователя"
// @Param 		 email query string true "email пользователя"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Переданы некорректные параметры"
// @Failure      401  {object} dto.ErrorResponse "Неавторизован"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/resend [post]
func (h *OtpHandler) ResendOTP(c *gin.Context) {
	id := c.DefaultQuery("id", "")
	email := c.DefaultQuery("email", "")

	if id == "" || email == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectUUIDError})
		return
	}

	err = h.osc.SendOTP(userID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка генерации OTP-кода: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

// * Вспомогательные функции

func (h *OtpHandler) handleLoginAction(c *gin.Context, user *models.User) {
	if user.ToBeDeletedAt != nil {
		recoveryToken, err := utils.GenerateTempToken(user.ID, 15*time.Minute, "recovery_token")
		if err != nil {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка recovery_token: " + err.Error()})
			return
		}
		// возвращем токен для восстановления аккаунта
		c.JSON(http.StatusOK, dto.RecoveryResponse{
			RecoveryToken: recoveryToken,
			ToBeDeletedAt: *user.ToBeDeletedAt,
		})
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	device := utils.ParseDeviceInfo(userAgent)
	access, refresh, _ := h.tsc.GenerateTokens(user.ID, ip, userAgent, device)

	utils.SetAuthCookies(c, refresh)

	c.JSON(http.StatusOK, dto.LoginResponse{
		UserID:      user.ID,
		Email:       user.Email,
		AccessToken: access,
	})
}

func (h *OtpHandler) handleChangeMailAction(c *gin.Context, user *models.User, email *string) {
	if email == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": Почта"})
		return
	}

	user.Email = *email
	if err := h.asc.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка обновления: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

func (h *OtpHandler) handleChangePasswordAction(c *gin.Context, user *models.User, password *string) {
	if password == nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError + ": Новый пароль"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	user.Password = string(hash)
	if err := h.asc.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка обновления: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, true)
}

func (h *OtpHandler) handleDeleteAccountAction(c *gin.Context, user *models.User) {
	deletionTime := time.Now().Add(3 * 24 * time.Hour)
	err := h.asc.ScheduleDeletion(user.ID, deletionTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка планирования удаления: " + err.Error()})
		return
	}

	_ = h.tsc.RevokeAllRefreshTokens(user.ID, nil)

	utils.ClearAuthCookies(c)

	c.JSON(http.StatusOK, true)
}
