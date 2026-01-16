package main

import (
	"auth/models"
	"auth/utils"
	"common/config"
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	repo *Repository
}

func NewAuthService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ! User

func (s *Service) Register(email, password string) (uuid.UUID, error) {
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

func (s *Service) Login(email, password string) (uuid.UUID, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil || !user.IsVerified {
		return uuid.Nil, errors.New("invalid credentials")
	}
	return user.ID, err
}

func (s *Service) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *Service) DeleteUserByID(id uuid.UUID) error {
	return s.repo.DeleteUser(id, true)
}

func (s *Service) UpdateUser(user *models.User) error {
	return s.repo.UpdateUser(user)
}

func (s *Service) UpdatePassword(userID uuid.UUID, newPassword string) error {
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

// ! Token

func (s *Service) GenerateTokens(userID uuid.UUID, ip, userAgent, device string) (string, string, error) {
	now := time.Now()
	// Access token
	access := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":  userID,
			"exp": now.Add(config.AppConfig.AccessTokenDuration).Unix(),
			"jti": uuid.New().String(), // id токена
		},
	)
	accessToken, err := access.SignedString(config.AppConfig.JWTSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshToken := uuid.New().String()
	expiresAt := now.Add(config.AppConfig.RefreshTokenDuration)

	if err := s.repo.CreateRefreshToken(refreshToken, userID, expiresAt, ip, userAgent, device); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) Refresh(refreshToken string) (string, string, error) {
	rt, err := s.repo.FindValidByToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid or expired token")
	}

	// Отзываем старый
	err = s.repo.Revoke(refreshToken)

	access, newRefresh, err := s.GenerateTokens(rt.UserID, rt.IP, rt.UserAgent, rt.Device)
	return access, newRefresh, err
}

func (s *Service) BlacklistAccessToken(c *gin.Context, accessToken string) error {
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
		key := "blacklist:access:" + jti
		config.AuthRedisClient().Set(c.Request.Context(), key, "revoked", remaining)
	}

	return nil
}

func (s *Service) RevokeRefreshToken(refreshToken string) error {
	return s.repo.Revoke(refreshToken)
}

func (s *Service) RevokeAllRefreshTokens(userID uuid.UUID) error {
	return s.repo.RevokeAll(userID)
}

func (s *Service) ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error) {
	return s.repo.ListActiveSessions(userID)
}

// ! OTP

func (s *Service) SendOTP(userID uuid.UUID, email string) (string, time.Time, error) {
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
