package service

import (
	"chat/internal/models"
	"chat/internal/models/dto"
	"chat/internal/repository"
	"errors"

	"github.com/google/uuid"
)

type MsgService interface {
	GetMessages(userID, chatID uuid.UUID, limit, offset int) ([]models.Message, int64, error)
	GetLastMessage(userID, chatID uuid.UUID) (*models.Message, error)

	CreateMessage(chatID uuid.UUID, userID *uuid.UUID, req *dto.MsgCreateRequest) (*models.Message, error)
	UpdateMessage(msgID, userID uuid.UUID, req *dto.MsgUpdateRequest) error
	DeleteMessage(msgID, userID uuid.UUID) error
}

type msgService struct {
	repo  repository.MsgRepository
	pRepo repository.ParticipantRepository
}

func NewMsgService(repo repository.MsgRepository, pRepo repository.ParticipantRepository) MsgService {
	return &msgService{repo: repo, pRepo: pRepo}
}

func (s *msgService) GetMessages(userID, chatID uuid.UUID, limit, offset int) ([]models.Message, int64, error) {
	isMember := s.pRepo.IsParticipant(userID, chatID)
	if !isMember {
		return nil, 0, errors.New("вы не являетесь участником чата")
	}

	msgs, total, err := s.repo.GetByChatID(chatID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return msgs, total, nil
}

func (s *msgService) GetLastMessage(userID, chatID uuid.UUID) (*models.Message, error) {
	isMember := s.pRepo.IsParticipant(userID, chatID)
	if !isMember {
		return nil, errors.New("вы не являетесь участником чата")
	}

	return s.repo.GetLastMessage(chatID)
}

func (s *msgService) CreateMessage(chatID uuid.UUID, userID *uuid.UUID, req *dto.MsgCreateRequest) (*models.Message, error) {
	if req.Content == "" || req.Type == "" {
		return nil, errors.New("invalid parameter in body of request")
	} else if len(req.Content) > 4096 {
		return nil, errors.New("content too long")
	}

	msg := &models.Message{
		ChatID:         chatID,
		UserID:         userID,
		Content:        req.Content,
		Type:           req.Type,
		ReplyToMessage: req.ReplyToMessage,
	}
	err := s.repo.SendMessage(msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *msgService) UpdateMessage(msgID, userID uuid.UUID, req *dto.MsgUpdateRequest) error {
	if req.Content == "" {
		return errors.New("content is required")
	}

	msg := &models.Message{
		ID:             msgID,
		UserID:         &userID,
		Content:        req.Content,
		ReplyToMessage: req.ReplyToMessage,
	}

	return s.repo.EditMessage(msg)
}

func (s *msgService) DeleteMessage(msgID, userID uuid.UUID) error {
	return s.repo.DeleteMessage(msgID, userID)
}
