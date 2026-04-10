package service

import (
	smtp "auth/internal/email"
	"auth/internal/models"
	"auth/internal/repository"
	"auth/pkg/redis"
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(email, password string) error
	VerifyNewAccount(token string) (*models.User, error)
	Login(email, password string) (uuid.UUID, error)
	GetUserByID(id uuid.UUID) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	DeleteUserByID(id uuid.UUID) error
	UpdateUser(user *models.User) error
	UpdatePassword(userID uuid.UUID, newPassword string) error
	ScheduleDeletion(userID uuid.UUID, deletionTime time.Time) error
	CancelDeletion(userID uuid.UUID) error
}
type authService struct {
	repo repository.AuthRepository
}

func NewAuthService(repo repository.AuthRepository) AuthService {
	return &authService{repo: repo}
}

// ! ВХОД В АККАУНТ

func (s *authService) Register(email, password string) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	user, err := s.repo.CreateUser(email, string(hash))
	if err != nil {
		return err
	}

	verificationToken := uuid.NewString()
	fmt.Println("Отправляем письмо пользовтелю: ", user.ID)
	err = redis.AuthRedis.Set(context.Background(), "verify:"+verificationToken, user.ID.String(), 15*time.Minute).Err()
	if err != nil {
		return err
	}

	go func() {
		err = smtp.SendVerificationEmail(email, verificationToken)
		if err != nil {
			log.Printf("Failed to send verification email to %s: %v", email, err)
		}
	}()

	return nil
}

func (s *authService) VerifyNewAccount(token string) (*models.User, error) {
	id, err := redis.AuthRedis.Get(context.Background(), "verify:"+token).Result()
	if err != nil {
		return nil, fmt.Errorf("invalid or expired verification token")
	}
	fmt.Println("Получен id пользователя: ", id)
	userID := uuid.MustParse(id)

	user, err := s.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	if user.IsVerified {
		return user, nil
	}

	user.IsVerified = true
	if err = s.repo.VerifyNewUser(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(email, password string) (uuid.UUID, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, errors.New("invalid credentials") // не говорим, что email не существует
		}
		return uuid.Nil, err
	}

	if !user.IsVerified {
		return uuid.Nil, errors.New("account is not verified")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return uuid.Nil, errors.New("invalid credentials")
	}

	return user.ID, nil
}

// ! ИНФОРМАЦИЯ О ПОЛЬЗОВАТЕЛЕ

func (s *authService) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *authService) GetUserByEmail(email string) (*models.User, error) {
	return s.repo.FindByEmail(email)
}

// ! ДЕЙСТВИЯ С ПОЛЬЗОВАТЕЛЕМ

func (s *authService) DeleteUserByID(id uuid.UUID) error {
	return s.repo.DeleteUser(id, true)
}

func (s *authService) UpdateUser(user *models.User) error {
	return s.repo.UpdateUser(user)
}

func (s *authService) UpdatePassword(userID uuid.UUID, newPassword string) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(newPassword)) == nil {
		return errors.New("пароли не должны совпадать")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	user.Password = string(hash)

	return s.repo.UpdateUser(user)
}

func (s *authService) ScheduleDeletion(userID uuid.UUID, deletionTime time.Time) error {
	return s.repo.ScheduleDeletion(userID, deletionTime)
}

func (s *authService) CancelDeletion(userID uuid.UUID) error {
	return s.repo.CancelDeletion(userID)
}
