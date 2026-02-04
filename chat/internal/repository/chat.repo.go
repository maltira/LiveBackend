package repository

import (
	"chat/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatRepository interface {
	IsChatExists(chatID uuid.UUID) (*models.Chat, error)
	GetAllChats(userID uuid.UUID) ([]models.Chat, error)
	GetByID(chatID uuid.UUID) (*models.Chat, error)

	CreatePrivateChat(user1ID, user2ID uuid.UUID) (*models.Chat, error)
	CreateGroupChat(name string, avatarURL *string, canJoin bool, ownerID uuid.UUID, members []uuid.UUID) (*models.Chat, error)
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db}
}

func (r *chatRepository) IsChatExists(chatID uuid.UUID) (*models.Chat, error) {
	var chat *models.Chat
	err := r.db.Where("chat_id = ?", chatID).First(&chat).Error
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) GetAllChats(userID uuid.UUID) ([]models.Chat, error) {
	var chats []models.Chat
	err := r.db.
		Joins("JOIN chat_participants ON chat_participants.chat_id = chats.id").
		Where("chat_participants.user_id = ?", userID).
		Order("chats.last_message_at DESC").
		Find(&chats).Error
	if err != nil {
		return nil, err
	}
	return chats, err
}

func (r *chatRepository) GetByID(chatID uuid.UUID) (*models.Chat, error) {
	var chat *models.Chat
	err := r.db.Where("chat_id = ?", chatID).First(&chat).Error
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (r *chatRepository) CreatePrivateChat(user1ID, user2ID uuid.UUID) (*models.Chat, error) {
	// * Ищем существующий чат
	var existing models.Chat
	err := r.db.
		Joins("JOIN chat_participants p1 ON p1.chat_id = chats.id").
		Joins("JOIN chat_participants p2 ON p2.chat_id = chats.id").
		Where("chats.type = ? AND p1.user_id = ? AND p2.user_id = ? AND p1.user_id != p2.user_id", "private", user1ID, user2ID).
		First(&existing).Error
	if err == nil {
		return &existing, nil
	}

	// * Создаём новый чат
	chat := &models.Chat{ // новый чат
		Type:      "private",
		CreatedBy: user1ID,
	}
	err = r.db.Transaction(func(tx *gorm.DB) error {
		if err = tx.Create(&chat).Error; err != nil {
			return err
		}

		participants := []models.Participant{ // добавляем учатсников
			{ChatID: chat.ID, UserID: user1ID, Role: "member", JoinedAt: time.Now()},
			{ChatID: chat.ID, UserID: user2ID, Role: "member", JoinedAt: time.Now()},
		}
		if err = tx.Create(&participants).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chat, nil
}

func (r *chatRepository) CreateGroupChat(name string, avatarURL *string, canJoin bool, ownerID uuid.UUID, members []uuid.UUID) (*models.Chat, error) {

	chat := &models.Chat{
		Type:      "group",
		CreatedBy: ownerID,
		Name:      &name,
		AvatarURL: avatarURL,
		CanJoin:   &canJoin,
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(chat).Error; err != nil {
			return err
		}

		var participants []models.Participant
		for _, member := range members {
			participants = append(participants, models.Participant{
				ChatID:   chat.ID,
				UserID:   member,
				Role:     "member",
				JoinedAt: time.Now(),
			})
		}
		if err := tx.Create(&participants).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return chat, nil
}
