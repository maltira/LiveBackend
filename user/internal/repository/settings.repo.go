package repository

import (
	"user/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SettingsRepository interface {
	GetSettings(profileID uuid.UUID) (*models.Settings, error)
	SaveSettings(userID uuid.UUID, updates map[string]interface{}) error
}

type settingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) GetSettings(profileID uuid.UUID) (*models.Settings, error) {
	settings := models.Settings{}
	err := r.db.Where("profile_id = ?", profileID).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *settingsRepository) SaveSettings(userID uuid.UUID, updates map[string]interface{}) error {
	return r.db.
		Model(&models.Settings{}).
		Where("profile_id = ?", userID).
		Updates(updates).Error
}
