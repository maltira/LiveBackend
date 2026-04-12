package handler

import (
	"auth/config"
	"auth/internal/dto"
	"auth/internal/models"
	"auth/internal/service"
	"auth/pkg/utils"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
