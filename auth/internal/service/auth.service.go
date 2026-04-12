package service

import (
	"auth/config"
	smtp "auth/internal/email"
	"auth/internal/models"
	"auth/internal/repository"
	"auth/pkg/redis"
	"auth/pkg/utils"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(email, password string) error
	VerifyNewAccount(token string) (*models.User, *gorm.DB, error)
	Login(email, password string) (uuid.UUID, error)
	FindByID(id uuid.UUID) (*models.User, error)
	FindByEmail(email string) (*models.User, error)

	SoftDeleteUserByID(id uuid.UUID, email, reason, deletedBy string) error
	UpdateUser(user *models.User) error
	ResetPassword(token, newPassword string) error
}
type authService struct {
	repo  repository.AuthRepository
	tRepo repository.TokenRepository
}

func NewAuthService(repo repository.AuthRepository, tRepo repository.TokenRepository) AuthService {
	return &authService{repo: repo, tRepo: tRepo}
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
		verifyURL := fmt.Sprintf("%s/verify-email?token=%s", config.Env.FrontendURL, verificationToken)
		err = smtp.SendVerificationEmail(email, verifyURL)
		if err != nil {
			log.Printf("Failed to send verification email to %s: %v", email, err)
		}
	}()

	return nil
}

func (s *authService) VerifyNewAccount(token string) (*models.User, *gorm.DB, error) {
	id, err := redis.AuthRedis.Get(context.Background(), "verify:"+token).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("invalid or expired verification token")
	}
	userID := uuid.MustParse(id)

	user, err := s.FindByID(userID)
	if err != nil {
		return nil, nil, err
	}

	if user.IsVerified {
		return user, nil, nil
	}

	user.IsVerified = true
	tx, err := s.repo.VerifyNewUser(user)
	if err != nil {
		return nil, nil, err
	}

	return user, tx, nil
}

func (s *authService) Login(email, password string) (uuid.UUID, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil || !user.IsVerified {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, errors.New("invalid credentials") // не говорим, что email не существует
		}
		return uuid.Nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return uuid.Nil, errors.New("invalid credentials")
	}

	return user.ID, nil
}

// ! ИНФОРМАЦИЯ О ПОЛЬЗОВАТЕЛЕ

func (s *authService) FindByID(id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *authService) FindByEmail(email string) (*models.User, error) {
	return s.repo.FindByEmail(email)
}

// ! ДЕЙСТВИЯ С ПОЛЬЗОВАТЕЛЕМ

func (s *authService) SoftDeleteUserByID(id uuid.UUID, email, reason, deletedBy string) error {
	return s.repo.SoftDeleteUserByID(id, email, reason, deletedBy)
}

func (s *authService) UpdateUser(user *models.User) error {
	return s.repo.UpdateUser(user)
}

func (s *authService) ResetPassword(token, newPassword string) error {
	ctx := context.Background()

	// Получаем все ключи reset:password:*
	keys, err := redis.AuthRedis.Keys(ctx, "reset:password:*").Result()
	if err != nil {
		return err
	}

	var userID uuid.UUID
	for _, key := range keys {
		hashedToken, _ := redis.AuthRedis.Get(ctx, key).Result()
		if utils.CompareToken(token, hashedToken) {
			userIDStr := strings.TrimPrefix(key, "reset:password:")
			userID = uuid.MustParse(userIDStr)
			break
		}
	}

	if userID == uuid.Nil {
		return errors.New("invalid or expired token")
	}

	// ищем пользователя и меняем пароль
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	if err = s.repo.UpdateUser(user); err != nil {
		return err
	}

	// Удаляем использованный токен
	redis.AuthRedis.Del(ctx, "reset:password:"+userID.String())

	// Отзываем все refresh-токены пользователя
	_ = s.tRepo.RevokeAll(user.ID, nil)

	return nil
}
