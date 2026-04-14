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
	VerifyNewUser(user *models.User) (*gorm.DB, error)

	SoftDeleteUserByID(id uuid.UUID, email, reason, deletedBy string) error
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
	var user *models.User

	err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *authRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ? AND deleted_at IS NULL", id).Error
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
				return nil, err
			}
		} else {
			return nil, errors.New("email already exists")
		}
	}
	return user, result.Error
}

func (r *authRepository) VerifyNewUser(user *models.User) (*gorm.DB, error) {
	tx := r.db.Begin()

	if err := tx.Save(&user).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (r *authRepository) SoftDeleteUserByID(id uuid.UUID, email, reason, deletedBy string) error {
	now := time.Now()
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"email":           "delete/" + now.Format("2006-01-02 15:04:05") + "/" + email,
		"deletion_reason": reason,
		"deleted_by":      deletedBy,
		"deleted_at":      time.Now(),
	}).Error
}

func (r *authRepository) UpdateUser(user *models.User) error {
	return r.db.Save(&user).Error
}
