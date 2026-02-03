package service

import (
	"errors"
	"time"
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
	user, err := sc.repo.FindByID(userID)
	var flag = false

	if err != nil {
		return errors.New("user not found")
	}

	if profile.Username != "" && user.Username != profile.Username {
		err = sc.repo.UsernameExists(user.Username)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user.Username = profile.Username
			flag = true
		} else {
			return errors.New("это имя пользователя занято")
		}
	}
	if profile.FullName != "" && user.FullName != profile.FullName {
		user.FullName = profile.FullName
		flag = true
	}
	if profile.Bio != user.Bio {
		user.Bio = profile.Bio
		flag = true
	}
	if profile.AvatarURL != user.AvatarURL {
		user.AvatarURL = profile.AvatarURL
		flag = true
	}

	now := time.Now()
	if profile.BirthDate == nil {
		user.BirthDate = nil
	} else if (*profile.BirthDate).Before(now) {
		user.BirthDate = profile.BirthDate
	} else {
		return errors.New("некорректная дата рождения")
	}

	if flag {
		return sc.repo.Update(user)
	}
	return errors.New("no columns to update")
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
