package service

import (
	"user/internal/models"
	"user/internal/repository"

	"github.com/google/uuid"
)

type SettingsService interface {
	GetSettings(profileID uuid.UUID) (*models.Settings, error)
	UpdateVisibleStatus(userID uuid.UUID, isVisible bool) error
	UpdateVisibleBirthDate(userID uuid.UUID, visible string) error
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
func (s *settingsService) UpdateVisibleStatus(userID uuid.UUID, isVisible bool) error {
	return s.repo.UpdateVisibleStatus(userID, isVisible)
}
func (s *settingsService) UpdateVisibleBirthDate(userID uuid.UUID, visible string) error {
	return s.repo.UpdateVisibleBirthDate(userID, visible)
}
