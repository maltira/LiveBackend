package repository

import (
	"chat/internal/models"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MsgRepository interface {
	GetByChatID(chatID uuid.UUID, limit, offset int) ([]models.Message, int64, error)
	GetLastMessage(chatID uuid.UUID) (*models.Message, error)

	SendMessage(msg *models.Message) error
	EditMessage(msg *models.Message) error
	DeleteMessage(msgID, userID uuid.UUID) error
}

type msgRepository struct {
	db *gorm.DB
}

func NewMsgRepository(db *gorm.DB) MsgRepository {
	return &msgRepository{db}
}

func (r *msgRepository) GetByChatID(chatID uuid.UUID, limit, offset int) ([]models.Message, int64, error) {
	var msgs []models.Message
	var count int64

	r.db.Model(&models.Message{}).Where("chat_id = ?", chatID).Count(&count)

	err := r.db.
		Where("chat_id = ?", chatID).
		Order("created_at desc").
		Limit(limit).
		Offset(offset).
		Find(&msgs).Error
	if err != nil {
		return nil, 0, err
	}

	return msgs, count, nil
}

func (r *msgRepository) GetLastMessage(chatID uuid.UUID) (*models.Message, error) {
	var msg models.Message
	err := r.db.
		Where("chat_id = ?", chatID).
		Order("created_at desc").
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (r *msgRepository) SendMessage(msg *models.Message) error {
	return r.db.Create(&msg).Error
}

func (r *msgRepository) EditMessage(msg *models.Message) error {
	if msg.ID == uuid.Nil {
		return errors.New("message ID is required")
	}

	updates := map[string]interface{}{
		"content":          msg.Content,
		"reply_to_message": *msg.ReplyToMessage,
		"edited_at":        time.Now(),
	}

	return r.db.
		Model(&models.Message{}).
		Where("id = ? AND user_id = ?", msg.ID, *msg.UserID).
		Updates(&updates).Error
}

func (r *msgRepository) DeleteMessage(msgID, userID uuid.UUID) error {
	return r.db.Where("id = ? AND user_id = ?", msgID, userID).Delete(&models.Message{}).Error
}
