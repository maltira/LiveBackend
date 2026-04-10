package repository

import (
	"auth/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TokenRepository interface {
	SaveRefreshToken(token string, userID uuid.UUID, expiresAt time.Time, ip, userAgent, device string) error
	FindRefreshToken(token string) (*models.RefreshToken, error)
	Revoke(token string) error
	RevokeAll(userID uuid.UUID, excludeToken *string) error
	ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error)
}
type tokenRepository struct {
	db *gorm.DB
}

func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) SaveRefreshToken(token string, userID uuid.UUID, expiresAt time.Time, ip, userAgent, device string) error {
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

func (r *tokenRepository) FindRefreshToken(token string) (*models.RefreshToken, error) {
	var rt *models.RefreshToken
	if err := r.db.Where("token = ?", token).First(&rt).Error; err != nil {
		return nil, err
	}

	return rt, nil
}

func (r *tokenRepository) Revoke(token string) error {
	return r.db.Where("token = ?", token).Delete(&models.RefreshToken{}).Error
}

func (r *tokenRepository) RevokeAll(userID uuid.UUID, excludeToken *string) error {
	query := r.db.Where("user_id = ?", userID)
	if excludeToken != nil {
		query = query.Where("token != ?", excludeToken)
	}
	return query.Delete(&models.RefreshToken{}).Error
}

func (r *tokenRepository) ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error) {
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
