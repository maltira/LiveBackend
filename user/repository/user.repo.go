package repository

import (
	"user/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.Profile) error
	FindByID(userID uuid.UUID) (*models.Profile, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.Profile) error {
	return r.db.Create(user).Error
}

func (r *userRepository) FindByID(userID uuid.UUID) (*models.Profile, error) {
	var user models.Profile
	err := r.db.First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
