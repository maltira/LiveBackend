package service

import (
	"errors"
	"user/internal/models"
	"user/internal/repository"
	"user/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlockService interface {
	GetAllBlocks(userID uuid.UUID) ([]models.Block, error)
	IsBlock(ID uuid.UUID, targetID uuid.UUID) (bool, error)

	BlockUser(userID uuid.UUID, blockedUserID uuid.UUID) (*models.Block, error)
	UnblockUser(userID uuid.UUID, blockedUserID uuid.UUID) error
}

type blockService struct {
	repo repository.BlockRepository
}

func NewBlockService(repo repository.BlockRepository) BlockService {
	return &blockService{repo: repo}
}

func (sc *blockService) GetAllBlocks(userID uuid.UUID) ([]models.Block, error) {
	return sc.repo.GetAllBlocks(userID)
}

func (sc *blockService) IsBlock(ID uuid.UUID, targetID uuid.UUID) (bool, error) {
	err := sc.repo.CheckBlock(ID, targetID)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (sc *blockService) BlockUser(userID uuid.UUID, blockedUserID uuid.UUID) (*models.Block, error) {
	if userID == blockedUserID {
		return nil, errors.New("нельзя заблокировать свой профиль")
	}

	var profile *models.Profile
	err := database.GetDB().First(&profile, "id = ?", blockedUserID).Error
	if err != nil {
		return nil, err
	}

	var block = models.Block{
		ProfileID:        userID,
		BlockedProfileID: blockedUserID,
	}
	err = sc.repo.BlockUser(&block)
	if err != nil {
		return nil, err
	}
	block.BlockedProfile = *profile

	return &block, nil
}

func (sc *blockService) UnblockUser(userID uuid.UUID, blockedUserID uuid.UUID) error {
	if userID == blockedUserID {
		return errors.New("нельзя разблокировать свой профиль")
	}
	return sc.repo.UnblockUser(userID, blockedUserID)
}
