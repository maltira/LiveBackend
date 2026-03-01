package service

import (
	"errors"
	"user/internal/models"
	"user/internal/models/dto"
	"user/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileService interface {
	Update(userID uuid.UUID, profile *dto.UpdateProfileRequest) error

	GetAll() ([]models.Profile, error)
	GetAllBySearch(search string, limit int) ([]models.Profile, error)
	FindByID(userID uuid.UUID) (*models.Profile, error)
	IsUsernameFree(username string) (bool, error)
}

type profileService struct {
	repo repository.ProfileRepository
}

func NewProfileService(repo repository.ProfileRepository) ProfileService {
	return &profileService{repo: repo}
}

func (sc *profileService) Update(userID uuid.UUID, profile *dto.UpdateProfileRequest) error {
	updates := make(map[string]interface{})

	if profile.Username != nil {
		err := sc.repo.UsernameExists(*profile.Username)
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			updates["username"] = *profile.Username
		} else {
			return errors.New("это имя пользователя занято")
		}
	}
	if profile.FullName != nil {
		updates["full_name"] = *profile.FullName
	}
	if profile.Bio != nil {
		updates["bio"] = *profile.Bio
	}
	if profile.AvatarURL != nil {
		updates["avatar_url"] = *profile.AvatarURL
	}

	if profile.BirthDate != nil {
		updates["birth_date"] = *profile.BirthDate
	} else if profile.BirthDateIsSet == true {
		updates["birth_date"] = nil
	}

	if len(updates) == 0 {
		return errors.New("no columns to update")
	}
	return sc.repo.Update(userID, updates)
}

func (sc *profileService) GetAll() ([]models.Profile, error) {
	return sc.repo.GetAll()
}
func (sc *profileService) GetAllBySearch(search string, limit int) ([]models.Profile, error) {
	return sc.repo.GetAllBySearch(search, limit)
}
func (sc *profileService) FindByID(userID uuid.UUID) (*models.Profile, error) {
	return sc.repo.FindByID(userID)
}
func (sc *profileService) IsUsernameFree(username string) (bool, error) {
	err := sc.repo.UsernameExists(username)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	return false, nil
}
