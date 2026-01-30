package repository

import (
	"time"
	"user/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Create(user *models.Profile) error
	Update(user *models.Profile) error

	GetAll() ([]models.Profile, error)
	FindByID(userID uuid.UUID) (*models.Profile, error)
	UsernameExists(username string) error

	UpdateLastSeen(userID uuid.UUID, lastSeen time.Time) error
}

type profileRepository struct {
	db *gorm.DB
}

func NewProfileRepository(db *gorm.DB) ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Create(user *models.Profile) error {
	return r.db.Create(user).Error
}
func (r *profileRepository) Update(user *models.Profile) error {
	return r.db.Save(user).Error
}

func (r *profileRepository) GetAll() ([]models.Profile, error) {
	var users []models.Profile
	err := r.db.Preload("Settings").Find(&users).Error
	return users, err
}
func (r *profileRepository) FindByID(userID uuid.UUID) (*models.Profile, error) {
	var user models.Profile
	err := r.db.Preload("Settings").First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *profileRepository) UsernameExists(username string) error {
	return r.db.First(&models.Profile{}, "username = ?", username).Error
}

func (r *profileRepository) UpdateLastSeen(userID uuid.UUID, lastSeen time.Time) error {
	return r.db.Model(&models.Profile{}).
		Where("id = ?", userID).
		Update("last_seen", lastSeen).Error
}
