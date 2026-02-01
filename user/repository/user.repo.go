package repository

import (
	"strings"
	"time"
	"user/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProfileRepository interface {
	Create(user *models.Profile) error
	Update(user *models.Profile) error

	GetAll() ([]models.Profile, error)
	GetAllBySearch(query string, limit int) ([]models.Profile, error)
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
func (r *profileRepository) GetAllBySearch(query string, limit int) ([]models.Profile, error) {
	if len(query) < 4 {
		return []models.Profile{}, nil
	}
	var users []models.Profile
	cleanQuery := strings.TrimSpace(strings.TrimPrefix(query, "@"))

	q := r.db.
		Where("username ILIKE ?", cleanQuery+"%").                    // префиксный поиск
		Order(gorm.Expr("similarity(username, ?) DESC", cleanQuery)). // сортировка по похожести
		Order("username ASC").                                        // затем по алфавиту
		Limit(limit)

	result := q.Preload("Settings").Find(&users)
	if result.Error != nil {
		return nil, result.Error
	}

	return users, nil
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
