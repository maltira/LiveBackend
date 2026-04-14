package repository

import (
	"auth/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OtpRepository interface {
	CreateOTP(otp *models.OTPCode) error
	FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error)
	MarkOTPAsUsed(id uuid.UUID) error
	InvalidateAllOTPs(userID uuid.UUID) error
}
type otpRepository struct {
	db *gorm.DB
}

func NewOtpRepository(db *gorm.DB) OtpRepository {
	return &otpRepository{db: db}
}

func (r *otpRepository) CreateOTP(otp *models.OTPCode) error {
	return r.db.Create(otp).Error
}

func (r *otpRepository) FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error) {
	var otp models.OTPCode
	err := r.db.
		Where("user_id = ? AND code = ? AND expires_at > ? AND is_used = false", userID, code, time.Now()).
		Order("created_at desc").
		First(&otp).Error
	if err != nil {
		return nil, err
	}
	return &otp, nil
}

func (r *otpRepository) MarkOTPAsUsed(id uuid.UUID) error {
	return r.db.Model(&models.OTPCode{}).
		Where("id = ?", id).
		Update("is_used", true).Error
}

func (r *otpRepository) InvalidateAllOTPs(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.OTPCode{}).Error
}
