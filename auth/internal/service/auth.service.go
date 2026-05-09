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
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(email, password string) error
	VerifyNewAccount(token string) error
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

func (s *authService) VerifyNewAccount(token string) error {
	id, err := redis.AuthRedis.Get(context.Background(), "verify:"+token).Result()
	if err != nil {
		return fmt.Errorf("invalid or expired verification token")
	}
	userID := uuid.MustParse(id)

	user, err := s.FindByID(userID)
	if err != nil {
		return err
	}

	if user.IsVerified {
		return nil
	}

	user.IsVerified = true
	tx, err := s.repo.VerifyNewUser(user)
	if err != nil {
		return err
	}

	// событие в очередь (3 попытки)
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		err = utils.SendRequestCreateProfile(userID)
		if err == nil {
			lastErr = nil
			break
		}

		lastErr = err
		log.Printf("Attempt %d/3 to create profile %s failed: %v", attempt, id, err)

		if attempt < 3 {
			delay := time.Duration(attempt*300) * time.Millisecond
			time.Sleep(delay)
		}
	}

	if lastErr != nil {
		tx.Rollback()
		return lastErr
	}

	redis.AuthRedis.Del(context.Background(), "verify:"+token)

	tx.Commit()
	return nil
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
	userIDStr, err := redis.AuthRedis.Get(ctx, "reset:password:"+token).Result()
	if err != nil {
		return errors.New("invalid or expired token")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return err
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
	redis.AuthRedis.Del(ctx, "reset:password:ref:"+userID.String())
	redis.AuthRedis.Del(ctx, "reset:password:"+token)

	// Отзываем все refresh-токены пользователя
	jtis, err := s.tRepo.GetActiveJTIs(userID, nil)
	if err != nil {
		log.Printf("Warning: failed to get JTIs for blacklist: %v", err)
	}
	for _, jti := range jtis {
		_ = BlacklistJTI(jti)
	}
	_ = s.tRepo.RevokeAll(user.ID, nil)

	return nil
}
