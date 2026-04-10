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

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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

	GenerateTokens(userID uuid.UUID, ip, userAgent, device string) (string, string, error)
	Refresh(refreshToken string) (string, string, error)
	BlacklistAccessToken(c *gin.Context, accessToken string) error
	RevokeRefreshToken(refreshToken string) error
	RevokeAllRefreshTokens(userID uuid.UUID) error
	ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error)

	MarkOTPAsUsed(id uuid.UUID) error
	SendOTP(userID uuid.UUID, email string) error
	FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error)
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

func (s *authService) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *authService) GetUserByEmail(email string) (*models.User, error) {
	return s.repo.FindByEmail(email)
}

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

// ! Token

func (s *authService) GenerateTokens(userID uuid.UUID, ip, userAgent, device string) (string, string, error) {
	now := time.Now()
	// Access token
	access := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":  userID,
			"exp": now.Add(config.Env.AccessTokenDuration).Unix(),
			"jti": uuid.New().String(), // id токена
		},
	)
	accessToken, err := access.SignedString(config.Env.JWTSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshToken := uuid.New().String()
	expiresAt := now.Add(config.Env.RefreshTokenDuration)

	if err = s.repo.CreateRefreshToken(refreshToken, userID, expiresAt, ip, userAgent, device); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *authService) Refresh(refreshToken string) (string, string, error) {
	rt, err := s.repo.FindValidByToken(refreshToken)
	if err != nil {
		return "", "", err
	}

	// Отзываем старый
	_ = s.repo.Revoke(refreshToken)

	access, newRefresh, err := s.GenerateTokens(rt.UserID, rt.IP, rt.UserAgent, rt.Device)
	return access, newRefresh, err
}

func (s *authService) BlacklistAccessToken(c *gin.Context, accessToken string) error {
	token, err := utils.ParseToken(accessToken)
	if err != nil || !token.Valid {
		return errors.New("invalid token")
	}

	claims, _ := token.Claims.(jwt.MapClaims)
	jti, _ := claims["jti"].(string)
	expFloat, _ := claims["exp"].(float64)
	exp := time.Unix(int64(expFloat), 0)

	remaining := time.Until(exp)
	if remaining > 0 {
		key := "auth:blacklist:access:" + jti
		redis.AuthRedis.Set(c.Request.Context(), key, "1", remaining)
	}

	return nil
}

func (s *authService) RevokeRefreshToken(refreshToken string) error {
	return s.repo.Revoke(refreshToken)
}

func (s *authService) RevokeAllRefreshTokens(userID uuid.UUID) error {
	return s.repo.RevokeAll(userID)
}

func (s *authService) ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error) {
	return s.repo.ListActiveSessions(userID)
}

// ! OTP

func (s *authService) SendOTP(userID uuid.UUID, email string) error {
	code := utils.GenerateOTP()
	expires := time.Now().Add(10 * time.Minute)

	otp := models.OTPCode{
		UserID:    userID,
		Code:      code,
		ExpiresAt: expires,
	}

	// инвалидируем прошлые коды, чтоб по ним нельзя было зайти
	if err := s.repo.InvalidateAllOTPs(userID); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}

	if err := s.repo.CreateOTP(&otp); err != nil {
		return err
	}

	go func() {
		err := smtp.SendOTP(email, code, expires.Format("15:04:05"))
		if err != nil {
			log.Printf("Failed to send OTP to %s: %v", email, err)
		}
	}()

	return nil
}

func (s *authService) FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error) {
	return s.repo.FindValidOTP(userID, code)
}

func (s *authService) MarkOTPAsUsed(id uuid.UUID) error {
	return s.repo.MarkOTPAsUsed(id)
}
