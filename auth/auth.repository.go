package main

import (
	"auth/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ! User

func (r *Repository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) DeleteUser(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}

func (r *Repository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *Repository) Logout(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}

// ! RefreshToken

func (r *Repository) CreateRefreshToken(token string, userID uuid.UUID, expiresAt time.Time) error {
	rt := models.RefreshToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	return r.db.Create(&rt).Error
}

func (r *Repository) FindValidByToken(token string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.
		Where("token = ? AND revoked = false AND expires_at > ?", token, time.Now()).
		First(&rt).Error
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *Repository) Revoke(token string) error {
	return r.db.Model(&models.RefreshToken{}).
		Where("token = ?", token).
		Update("revoked", true).Error
}

// ! OTP

func (r *Repository) CreateOTP(otp *models.OTPCode) error {
	return r.db.Create(otp).Error
}

func (r *Repository) FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error) {
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

func (r *Repository) MarkOTPAsUsed(id uuid.UUID) error {
	return r.db.Model(&models.OTPCode{}).
		Where("id = ?", id).
		Update("is_used", true).Error
}

func (r *Repository) InvalidateAllActiveOTPs(userID uuid.UUID) error {
	return r.db.Model(&models.OTPCode{}).
		Where("user_id = ? AND is_used = false AND expires_at > ?", userID, time.Now()).
		Update("is_used", true).Error
}
