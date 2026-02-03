package repository

import (
	"auth/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uuid.UUID) (*models.User, error)
	CreateUser(user *models.User) error
	DeleteUser(id uuid.UUID, isSoft bool) error
	ScheduleDeletion(userID uuid.UUID, deletionTime time.Time) error
	CancelDeletion(userID uuid.UUID) error
	UpdateUser(user *models.User) error

	CreateRefreshToken(token string, userID uuid.UUID, expiresAt time.Time, ip, userAgent, device string) error
	FindValidByToken(token string) (*models.RefreshToken, error)
	Revoke(token string) error
	RevokeAll(userID uuid.UUID) error
	ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error)

	CreateOTP(otp *models.OTPCode) error
	FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error)
	MarkOTPAsUsed(id uuid.UUID) error
	InvalidateAllActiveOTPs(userID uuid.UUID) error
}
type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

// ! User

func (r *authRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *authRepository) DeleteUser(id uuid.UUID, isSoft bool) error {
	if isSoft {
		return r.db.Delete(&models.User{}, "id = ?", id).Error
	}
	return r.db.Unscoped().Delete(&models.User{}, "id = ?", id).Error
}

func (r *authRepository) ScheduleDeletion(userID uuid.UUID, deletionTime time.Time) error {
	user, err := r.FindByID(userID)
	if err != nil {
		return err
	}
	user.ToBeDeletedAt = &deletionTime
	return r.db.Save(user).Error
}

func (r *authRepository) CancelDeletion(userID uuid.UUID) error {
	user, err := r.FindByID(userID)
	if err != nil {
		return err
	}
	user.ToBeDeletedAt = nil
	return r.db.Save(user).Error
}

func (r *authRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

// ! RefreshToken

func (r *authRepository) CreateRefreshToken(token string, userID uuid.UUID, expiresAt time.Time, ip, userAgent, device string) error {
	rt := models.RefreshToken{
		Token:     token,
		UserID:    userID,
		ExpiresAt: expiresAt,
		IP:        ip,
		UserAgent: userAgent,
		Device:    device,
	}
	return r.db.Create(&rt).Error
}

func (r *authRepository) FindValidByToken(token string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.
		Where("token = ? AND expires_at > ?", token, time.Now()).
		First(&rt).Error
	if err != nil {
		return nil, err
	}

	return &rt, nil
}

func (r *authRepository) Revoke(token string) error {
	return r.db.Where("token = ?", token).Delete(&models.RefreshToken{}).Error
}

func (r *authRepository) RevokeAll(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}

func (r *authRepository) ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error) {
	var sessions []models.RefreshToken
	err := r.db.
		Where("user_id = ? AND expires_at > ?", userID, time.Now()).
		Order("expires_at desc").
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

// ! OTP

func (r *authRepository) CreateOTP(otp *models.OTPCode) error {
	return r.db.Create(otp).Error
}

func (r *authRepository) FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error) {
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

func (r *authRepository) MarkOTPAsUsed(id uuid.UUID) error {
	return r.db.Model(&models.OTPCode{}).
		Where("id = ?", id).
		Update("is_used", true).Error
}

func (r *authRepository) InvalidateAllActiveOTPs(userID uuid.UUID) error {
	return r.db.Model(&models.OTPCode{}).
		Where("user_id = ? AND is_used = false AND expires_at > ?", userID, time.Now()).
		Update("is_used", true).Error
}
