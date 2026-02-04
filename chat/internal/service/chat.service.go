package service

import (
	"chat/internal/models"
	"chat/internal/models/dto"
	"chat/internal/repository"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ChatService interface {
	IsChatExists(chatID uuid.UUID) (bool, error)
	GetAllChats(userID uuid.UUID) ([]models.Chat, error)

	CreatePrivateChat(user1ID, user2ID uuid.UUID) (*models.Chat, error)
	CreateGroupChat(req *dto.ChatCreateRequest) (*models.Chat, error)
}

type chatService struct {
	repo repository.ChatRepository
}

func NewChatService(repo repository.ChatRepository) ChatService {
	return &chatService{repo: repo}
}

func (sc *chatService) IsChatExists(chatID uuid.UUID) (bool, error) {
	_, err := sc.repo.IsChatExists(chatID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (sc *chatService) GetAllChats(userID uuid.UUID) ([]models.Chat, error) {
	return sc.repo.GetAllChats(userID)
}

func (sc *chatService) CreatePrivateChat(user1ID, user2ID uuid.UUID) (*models.Chat, error) {
	return sc.repo.CreatePrivateChat(user1ID, user2ID)
}

func (sc *chatService) CreateGroupChat(req *dto.ChatCreateRequest) (*models.Chat, error) {
	if req.AvatarURL != nil && len(*req.AvatarURL) == 0 {
		return nil, errors.New("incorrect avatarURL")
	}
	return sc.repo.CreateGroupChat(req.Name, req.AvatarURL, req.CanJoin, req.CreatedBy, req.Members)
}
