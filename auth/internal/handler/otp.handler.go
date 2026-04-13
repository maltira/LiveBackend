package handler

import (
	"auth/config"
	"auth/internal/dto"
	"auth/internal/models"
	"auth/internal/service"
	"auth/pkg/utils"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type OtpHandler struct {
	osc service.OtpService
	asc service.AuthService
	tsc service.TokenService
}

func NewOtpHandler(osc service.OtpService, asc service.AuthService, tsc service.TokenService) *OtpHandler {
	return &OtpHandler{osc: osc, asc: asc, tsc: tsc}
}

// SendOTP
// @Summary      Отправка OTP
// @Description  Позволяет сгенерировать и отправить OTP-код
// @Tags         otp
// @Accept       json
// @Produce      json
// @Param 		 body body dto.SendOTPRequest true "UserID и Email"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Переданы некорректные параметры"
// @Router       /auth/otp/send [post]
func (h *OtpHandler) SendOTP(c *gin.Context) {
	var req dto.SendOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	err := h.osc.SendOTP(req.UserID, req.Email)
	if err != nil {
		log.Println("Ошибка генерации OTP:" + err.Error())
		// не прерываемся, тк пользователь сможет переотправить код
	}

	// отправляем на верификацию кода
	c.JSON(http.StatusOK, dto.OTPSentResponse{
		UserID:  req.UserID,
		Message: "OTP-код отправляен на указанную почту",
	})
}

// VerifyLoginOTP
// @Summary      Подтверждение входа
// @Description  Проверка OTP-кода + вход в аккаунт
// @Tags         otp
// @Accept       json
// @Produce      json
// @Param        body body dto.VerifyLoginOTPRequest true "UserID + Code"
// @Success      200  {object} dto.LoginResponse "Токен"
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/login/verify [put]
func (h *OtpHandler) VerifyLoginOTP(c *gin.Context) {
	var req dto.VerifyLoginOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	_, user, err := h.verifyAndMarkOTP(req.UserID, req.Code)
	if err != nil {
		h.handleOTPError(c, err)
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

// VerifyDeleteAccountOTP
// @Summary      Подтверждение удаления аккаунта
// @Description  Проверка OTP-кода + удаление аккаунта
// @Tags         otp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.VerifyDeleteOTPRequest true "UserID + Code + Reason"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/delete-account [post]
func (h *OtpHandler) VerifyDeleteAccountOTP(c *gin.Context) {
	var req dto.VerifyDeleteOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	_, user, err := h.verifyAndMarkOTP(req.UserID, req.Code)
	if err != nil {
		h.handleOTPError(c, err)
		return
	}

	err = h.asc.SoftDeleteUserByID(user.ID, user.Email, req.Reason, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка удаления пользователя: " + err.Error()})
		return
	}

	_ = h.tsc.RevokeAllRefreshTokens(user.ID, nil)
	utils.ClearAuthCookies(c)

	c.JSON(http.StatusOK, true)
}

// VerifyChangeMailOTP
// @Summary      Подтверждение смены почты
// @Description  Проверка OTP-кода + смена почты
// @Tags         otp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.VerifyChangeEmailRequest true "UserID + Code + NewEmail"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/change/email/verify [put]
func (h *OtpHandler) VerifyChangeMailOTP(c *gin.Context) {
	var req dto.VerifyChangeEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	_, user, err := h.verifyAndMarkOTP(req.UserID, req.Code)
	if err != nil {
		h.handleOTPError(c, err)
		return
	}

	user.Email = req.NewEmail
	user.EmailUpdatedAt = time.Now()
	if err = h.asc.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка обновления: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, true)
}

// VerifyChangePasswordOTP
// @Summary      Подтверждение смены пароля
// @Description  Проверка OTP-кода + смена пароля
// @Tags         otp
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.VerifyChangePassRequest true "UserID + Code + NewPassword"
// @Success      200  {boolean} true
// @Failure      400  {object} dto.ErrorResponse "Некорректные данные"
// @Failure      404  {object} dto.ErrorResponse "Запись не найдена"
// @Failure      500  {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router       /auth/change/pass/verify [put]
func (h *OtpHandler) VerifyChangePasswordOTP(c *gin.Context) {
	var req dto.VerifyChangePassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{Code: 400, Error: config.IncorrectDataError})
		return
	}

	_, user, err := h.verifyAndMarkOTP(req.UserID, req.Code)
	if err != nil {
		h.handleOTPError(c, err)
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	user.Password = string(hash)
	user.PasswordUpdatedAt = time.Now()
	if err = h.asc.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{Code: 500, Error: "Ошибка обновления: " + err.Error()})
		return
	}

	if err = h.tsc.RevokeAllRefreshTokens(user.ID, nil); err != nil {
		log.Printf("Warning: failed to revoke refresh tokens for user %s: %v", user.ID, err)
		// Не прерываем процесс удаления из-за ошибки с токенами
	}
	utils.ClearAuthCookies(c)

	c.JSON(http.StatusOK, true)
}

// ! Вспомогательные функции

// verifyAndMarkOTP - общая логика проверки и пометки OTP как использованного
func (h *OtpHandler) verifyAndMarkOTP(userID uuid.UUID, code string) (*models.OTPCode, *models.User, error) {
	// находим валидный OTP
	otp, err := h.osc.FindValidOTP(userID, code)
	if err != nil {
		return nil, nil, errors.New("invalid or expired otp")
	}

	// помечаем OTP как использованный
	if err = h.osc.MarkOTPAsUsed(otp.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to mark otp as used: %w", err)
	}

	// получаем пользователя
	user, err := h.asc.FindByID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("user not found: %w", err)
	}

	return otp, user, nil
}

// handleOTPError — централизованная обработка ошибок OTP
func (h *OtpHandler) handleOTPError(c *gin.Context, err error) {
	switch {
	case err.Error() == "invalid or expired otp":
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Code:  400,
			Error: config.InvalidOtpError,
		})
	case strings.Contains(err.Error(), "user not found"):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Code:  404,
			Error: config.NotFoundError + ": Пользователь",
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Code:  500,
			Error: "Ошибка обработки OTP: " + err.Error(),
		})
	}
}
