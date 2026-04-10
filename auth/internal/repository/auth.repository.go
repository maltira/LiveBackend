package repository

import (
	"auth/internal/models"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uuid.UUID) (*models.User, error)

	CreateUser(email, password string) (*models.User, error)
	VerifyNewUser(user *models.User) error

	DeleteUser(id uuid.UUID, isSoft bool) error
	ScheduleDeletion(userID uuid.UUID, deletionTime time.Time) error
	CancelDeletion(userID uuid.UUID) error
	UpdateUser(user *models.User) error
}
type authRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

// ! User

func (r *authRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User

	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *authRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) CreateUser(email, password string) (*models.User, error) {
	var user *models.User
	result := r.db.
		Where(models.User{Email: email}).
		Attrs(models.User{Password: password}).
		FirstOrCreate(&user)

	if result.RowsAffected == 0 {
		if !user.IsVerified {
			user.Password = password
			if err := r.db.Save(&user).Error; err != nil {
				return user, err
			}
		} else {
			return user, errors.New("email already exists")
		}
	}
	return user, result.Error
}

func (r *authRepository) VerifyNewUser(user *models.User) error {
	return r.db.Save(&user).Error
}

func (r *authRepository) DeleteUser(id uuid.UUID, isSoft bool) error {
	if isSoft {
		return r.db.Delete(&models.User{}, "id = ?", id).Error
	}
	return r.db.Unscoped().Delete(&models.User{}, "id = ?", id).Error
}

func (r *authRepository) ScheduleDeletion(userID uuid.UUID, deletionTime time.Time) error {
	user, err := r.FindByID(userID)
	if err != nil {
		return err
	}
	user.ToBeDeletedAt = &deletionTime
	return r.db.Save(user).Error
}

func (r *authRepository) CancelDeletion(userID uuid.UUID) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("to_be_deleted_at", nil).Error
}

func (r *authRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}
