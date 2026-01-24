package repository

import (
	"user/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SettingsRepository interface {
	GetSettings(profileID uuid.UUID) (*models.Settings, error)
	SaveSettings(settings *models.Settings) error
}

type settingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) GetSettings(profileID uuid.UUID) (*models.Settings, error) {
	settings := models.Settings{}
	err := r.db.Where("id = ?", profileID).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *settingsRepository) SaveSettings(settings *models.Settings) error {
	return r.db.Save(settings).Error
}
