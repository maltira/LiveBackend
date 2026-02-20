package service

import (
	"errors"
	"user/internal/models"
	"user/internal/models/dto"
	"user/internal/repository"

	"github.com/google/uuid"
)

type SettingsService interface {
	GetSettings(profileID uuid.UUID) (*models.Settings, error)
	SaveSettings(userID uuid.UUID, req *dto.SettingsUpdateRequest) error
}

type settingsService struct {
	repo repository.SettingsRepository
}

func NewSettingsService(repo repository.SettingsRepository) SettingsService {
	return &settingsService{repo: repo}
}

func (s *settingsService) GetSettings(profileID uuid.UUID) (*models.Settings, error) {
	return s.repo.GetSettings(profileID)
}
func (s *settingsService) SaveSettings(userID uuid.UUID, req *dto.SettingsUpdateRequest) error {
	updates := make(map[string]interface{})

	if req.DarkMode != nil {
		updates["dark_mode"] = *req.DarkMode
	}
	if req.ShowOnlineStatus != nil {
		updates["show_online_status"] = *req.ShowOnlineStatus
	}
	if req.ShowBirthDate != nil {
		updates["show_birth_date"] = *req.ShowBirthDate
	}
	if req.Language != nil {
		updates["language"] = *req.Language
	}

	if len(updates) == 0 {
		return errors.New("no settings to update")
	}
	return s.repo.SaveSettings(userID, updates)
}
