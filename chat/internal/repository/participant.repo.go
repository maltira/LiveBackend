package repository

import (
	"chat/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ParticipantRepository interface {
	GetParticipantByID(chatID, userID uuid.UUID) (*models.Participant, error)
	GetAllParticipants(chatID uuid.UUID) ([]models.Participant, error)
	IsParticipant(chatID, userID uuid.UUID) bool

	JoinToChat(chatID, userID uuid.UUID) error
	LeaveChat(chatID, userID uuid.UUID) error

	KickParticipant(chatID, kickedID uuid.UUID) error
	MuteParticipant(chatID, mutedID uuid.UUID, date time.Time) error
	UnmuteParticipant(chatID, unmutedID uuid.UUID) error
}

type participantRepository struct {
	db *gorm.DB
}

func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepository{db}
}

func (r *participantRepository) GetParticipantByID(chatID, userID uuid.UUID) (*models.Participant, error) {
	var participant *models.Participant
	err := r.db.Where("chat_id = ? AND user_id = ?", chatID, userID).First(&participant).Error
	if err != nil {
		return nil, err
	}
	return participant, nil
}

func (r *participantRepository) GetAllParticipants(chatID uuid.UUID) ([]models.Participant, error) {
	var participants []models.Participant
	err := r.db.Where("chat_id = ?", chatID).Find(&participants).Error
	if err != nil {
		return nil, err
	}
	return participants, nil
}

func (r *participantRepository) IsParticipant(chatID, userID uuid.UUID) bool {
	var count int64
	r.db.Model(&models.Participant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Count(&count)

	return count > 0
}

func (r *participantRepository) JoinToChat(chatID, userID uuid.UUID) error {
	return r.db.Create(&models.Participant{ChatID: chatID, UserID: userID}).Error
}

func (r *participantRepository) LeaveChat(chatID, userID uuid.UUID) error {
	return r.db.Delete(&models.Participant{}, "chat_id = ? AND user_id = ?", chatID, userID).Error
}

func (r *participantRepository) KickParticipant(chatID, kickedID uuid.UUID) error {
	return r.db.Delete(&models.Participant{}, "chat_id = ? AND user_id = ?", chatID, kickedID).Error
}

func (r *participantRepository) MuteParticipant(chatID, mutedID uuid.UUID, date time.Time) error {
	return r.db.
		Model(&models.Participant{}).
		Where("chat_id = ? AND user_id = ?", chatID, mutedID).Update("muted_until", date).Error
}

func (r *participantRepository) UnmuteParticipant(chatID, unmutedID uuid.UUID) error {
	return r.db.
		Model(&models.Participant{}).
		Where("chat_id = ? AND user_id = ?", chatID, unmutedID).
		Update("muted_until", nil).Error
}
