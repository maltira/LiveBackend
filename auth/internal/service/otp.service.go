package service

import (
	smtp "auth/internal/email"
	"auth/internal/models"
	"auth/internal/repository"
	"auth/pkg/utils"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OtpService interface {
	MarkOTPAsUsed(id uuid.UUID) error
	SendOTP(userID uuid.UUID, email string) error
	FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error)
}
type otpService struct {
	repo repository.OtpRepository
}

func NewOtpService(repo repository.OtpRepository) OtpService {
	return &otpService{repo: repo}
}

func (s *otpService) SendOTP(userID uuid.UUID, email string) error {
	code := utils.GenerateOTP()
	expires := time.Now().Add(10 * time.Minute)

	otp := models.OTPCode{
		UserID:    userID,
		Code:      code,
		ExpiresAt: expires,
	}

	// инвалидируем прошлые коды, чтоб по ним нельзя было зайти
	if err := s.repo.InvalidateAllOTPs(userID); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if err := s.repo.CreateOTP(&otp); err != nil {
		return err
	}

	go func() {
		err := smtp.SendOTP(email, code, expires.Format("15:04:05"))
		if err != nil {
			log.Printf("Failed to send OTP to %s: %v", email, err)
		}
	}()

	return nil
}

func (s *otpService) FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error) {
	return s.repo.FindValidOTP(userID, code)
}

func (s *otpService) MarkOTPAsUsed(id uuid.UUID) error {
	return s.repo.MarkOTPAsUsed(id)
}
