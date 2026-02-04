package service

import (
	"chat/internal/models"
	"chat/internal/repository"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ParticipantService interface {
	GetParticipantByID(chatID, userID uuid.UUID) (*models.Participant, error)
	IsParticipant(chatID, userID uuid.UUID) bool

	JoinToChat(chatID uuid.UUID, userID uuid.UUID) error
	LeaveChat(chatID uuid.UUID, userID uuid.UUID) error

	KickParticipant(userID, chatID, kickedID uuid.UUID) error
	MuteParticipant(userID, chatID, mutedID uuid.UUID, date time.Time) error
	UnmuteParticipant(userID, chatID, mutedID uuid.UUID) error
}

type participantService struct {
	repo     repository.ParticipantRepository
	chatRepo repository.ChatRepository
}

func NewParticipantService(repo repository.ParticipantRepository, chatRepo repository.ChatRepository) ParticipantService {
	return &participantService{repo: repo, chatRepo: chatRepo}
}

func (sc *participantService) GetParticipantByID(chatID, userID uuid.UUID) (*models.Participant, error) {
	return sc.repo.GetParticipantByID(chatID, userID)
}

func (sc *participantService) IsParticipant(chatID, userID uuid.UUID) bool {
	return sc.repo.IsParticipant(chatID, userID)
}

func (sc *participantService) JoinToChat(chatID, userID uuid.UUID) error {
	chat, err := sc.chatRepo.IsChatExists(chatID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("данный чат не существует")
		}
		return err
	} else if *chat.CanJoin {
		return sc.repo.JoinToChat(chatID, userID)
	}
	return errors.New("доступ к чату ограничен")
}

func (sc *participantService) LeaveChat(chatID, userID uuid.UUID) error {
	_, err := sc.chatRepo.IsChatExists(chatID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("данный чат не существует")
		}
		return err
	}
	return sc.repo.LeaveChat(chatID, userID)
}

func (sc *participantService) KickParticipant(userID, chatID, kickedID uuid.UUID) error {
	if userID == kickedID {
		return errors.New("невозможно исключить самого себя")
	}
	user, err := sc.GetParticipantByID(chatID, userID)
	if err != nil {
		return errors.New("не удалось получить информацию об админе")
	}
	kicked, err := sc.GetParticipantByID(chatID, kickedID)
	if err != nil {
		return errors.New("не удалось получить информацию о пользователе")
	}

	if user.Role == "owner" || user.Role == "admin" && kicked.Role == "member" {
		return sc.repo.KickParticipant(chatID, kickedID)
	}
	return errors.New("у вас недостаточно прав")
}

func (sc *participantService) MuteParticipant(userID, chatID, mutedID uuid.UUID, date time.Time) error {
	if userID == mutedID {
		return errors.New("невозможно исключить самого себя")
	}
	if date.Before(time.Now()) {
		return errors.New("указано некорретное время")
	}
	user, err := sc.GetParticipantByID(chatID, userID)
	if err != nil {
		return errors.New("не удалось получить информацию об админе")
	}
	muted, err := sc.GetParticipantByID(chatID, mutedID)
	if err != nil {
		return errors.New("не удалось получить информацию о пользователе")
	}

	if user.Role == "owner" || user.Role == "admin" && muted.Role == "member" {
		return sc.repo.MuteParticipant(chatID, mutedID, date)
	}
	return errors.New("у вас недостаточно прав")
}

func (sc *participantService) UnmuteParticipant(userID, chatID, mutedID uuid.UUID) error {
	if userID == mutedID {
		return errors.New("вы не можете размутить сами себя")
	}

	user, err := sc.GetParticipantByID(chatID, userID)
	if err != nil {
		return errors.New("не удалось получить информацию об админе")
	}

	if user.Role == "owner" || user.Role == "admin" {
		return sc.repo.UnmuteParticipant(chatID, mutedID)
	}
	return errors.New("у вас недостаточно прав")
}
