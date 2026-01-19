package repository

import (
	"user/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlockRepository interface {
	GetAllBlocks(userID uuid.UUID) ([]models.Block, error)
	CheckBlock(userID uuid.UUID, blockedUserID uuid.UUID) error

	BlockUser(Block *models.Block) error
	UnblockUser(userID, blockedUserID uuid.UUID) error
}

type blockRepository struct {
	db *gorm.DB
}

func NewBlockRepository(db *gorm.DB) BlockRepository {
	return &blockRepository{db: db}
}

func (r *blockRepository) GetAllBlocks(userID uuid.UUID) ([]models.Block, error) {
	var blocks []models.Block
	err := r.db.Where("profile_id = ?", userID).Preload("Profile").Find(&blocks).Error
	return blocks, err
}

func (r *blockRepository) CheckBlock(userID uuid.UUID, blockedUserID uuid.UUID) error {
	return r.db.First(&models.Block{}, "profile_id = ? AND blocked_profile_id = ?", userID, blockedUserID).Error
}

func (r *blockRepository) BlockUser(Block *models.Block) error {
	return r.db.Create(Block).Error
}

func (r *blockRepository) UnblockUser(userID, blockedUserID uuid.UUID) error {
	return r.db.
		Where("profile_id = ? AND blocked_profile_id = ?", userID, blockedUserID).
		Delete(&models.Block{}).Error
}
