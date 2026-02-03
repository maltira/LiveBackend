package service

import (
	"auth/config"
	"auth/internal/models"
	"auth/internal/repository"
	"auth/pkg/redis"
	"auth/pkg/utils"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService interface {
	Register(email, password string) (uuid.UUID, error)
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
	SendOTP(userID uuid.UUID, email string) (string, time.Time, error)
	FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error)
}
type authService struct {
	repo repository.AuthRepository
}

func NewAuthService(repo repository.AuthRepository) AuthService {
	return &authService{repo: repo}
}

// ! User

func (s *authService) Register(email, password string) (uuid.UUID, error) {
	res, err := s.repo.FindByEmail(email)
	if err == nil { // Такой пользователь есть
		if !res.IsVerified {
			delErr := s.repo.DeleteUser(res.ID, false)
			if delErr != nil {
				return uuid.Nil, errors.New("error deleting old unverified user")
			}
		} else {
			return uuid.Nil, errors.New("email already exists")
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, err
	}

	user := models.User{Email: email, Password: string(hash)}
	if err := s.repo.CreateUser(&user); err != nil {
		return uuid.Nil, err
	}

	return user.ID, nil
}

func (s *authService) Login(email, password string) (uuid.UUID, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil || !user.IsVerified {
		return uuid.Nil, errors.New("invalid credentials")
	}
	return user.ID, err
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

	if err := s.repo.CreateRefreshToken(refreshToken, userID, expiresAt, ip, userAgent, device); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *authService) Refresh(refreshToken string) (string, string, error) {
	rt, err := s.repo.FindValidByToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid or expired token")
	}

	// Отзываем старый
	err = s.repo.Revoke(refreshToken)

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

func (s *authService) SendOTP(userID uuid.UUID, email string) (string, time.Time, error) {
	code := utils.GenerateOTP()
	expires := time.Now().Add(10 * time.Minute)

	otp := models.OTPCode{
		UserID:    userID,
		Code:      code,
		ExpiresAt: expires,
	}

	if err := s.repo.InvalidateAllActiveOTPs(userID); err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", time.Time{}, err
		}
	}

	if err := s.repo.CreateOTP(&otp); err != nil {
		return "", time.Time{}, err
	}

	fmt.Printf("\n*| email: %s\n*| otp code: %s\n*| expires at: %s\n\n", email, otp.Code, expires.Format("15:04:05"))

	return code, expires, nil
}

func (s *authService) FindValidOTP(userID uuid.UUID, code string) (*models.OTPCode, error) {
	return s.repo.FindValidOTP(userID, code)
}

func (s *authService) MarkOTPAsUsed(id uuid.UUID) error {
	return s.repo.MarkOTPAsUsed(id)
}
