package service

import (
	"errors"
	"user/internal/models"
	"user/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileService interface {
	Create(userID uuid.UUID) error
	Update(userID uuid.UUID, data *map[string]interface{}) error

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

func (sc *profileService) Create(userID uuid.UUID) error {
	name := "user_" + userID.String()[:8]
	profile := &models.Profile{
		ID:        userID,
		Username:  name,
		FullName:  name,
		AvatarURL: "https://i.ibb.co/2Y0R1nDf/avatar-white.png",
	}
	settings := &models.Settings{
		ProfileID: userID,
	}
	return sc.repo.Create(profile, settings)
}

func (sc *profileService) Update(userID uuid.UUID, data *map[string]interface{}) error {
	updates := make(map[string]interface{})

	if v, ok := (*data)["username"]; ok {
		if len(v.(string)) < 4 || len(v.(string)) > 16 {
			return errors.New("username error")
		}

		err := sc.repo.UsernameExists(v.(string))
		if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
			updates["username"] = v
		} else {
			return errors.New("username exists")
		}
	}

	if v, ok := (*data)["full_name"]; ok {
		if len(v.(string)) < 4 || len(v.(string)) > 100 {
			return errors.New("full_name error")
		}
		updates["full_name"] = v
	}

	if v, ok := (*data)["bio"]; ok {
		if len(v.(string)) > 500 {
			return errors.New("bio error")
		}
		updates["bio"] = v
	}

	if v, ok := (*data)["avatar_url"]; ok {
		updates["avatar_url"] = v
	}

	if v, ok := (*data)["birth_date"]; ok {
		updates["birth_date"] = v
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
