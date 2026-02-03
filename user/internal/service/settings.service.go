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
	settings, err := s.repo.GetSettings(userID)
	if err != nil {
		return err
	}

	flag := false
	if req.DarkMode != settings.DarkMode {
		settings.DarkMode = req.DarkMode
		flag = true
	}
	if req.ShowOnlineStatus != settings.ShowOnlineStatus {
		settings.ShowOnlineStatus = req.ShowOnlineStatus
		flag = true
	}
	if req.ShowBirthDate != settings.ShowBirthDate {
		settings.ShowBirthDate = req.ShowBirthDate
		flag = true
	}
	if req.Language != settings.Language {
		settings.Language = req.Language
		flag = true
	}

	if !flag {
		return errors.New("no settings to update")
	}
	return s.repo.SaveSettings(settings)
}
