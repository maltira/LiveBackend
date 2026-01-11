package main

import (
	"auth/models"
	"auth/utils"
	"common/config"
	"errors"
	"fmt"
	"time"

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
			delErr := s.repo.DeleteUser(res.ID)
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
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return uuid.Nil, errors.New("invalid credentials")
	}
	return user.ID, err
}

func (s *Service) Logout(userID uuid.UUID) error {
	return s.repo.Logout(userID)
}

func (s *Service) GetUserByID(id uuid.UUID) (*models.User, error) {
	return s.repo.FindByID(id)
}

func (s *Service) DeleteUserByID(id uuid.UUID) error {
	return s.repo.DeleteUser(id)
}

func (s *Service) UpdateUser(user *models.User) error {
	return s.repo.UpdateUser(user)
}

// ! Token

func (s *Service) GenerateTokens(userID uuid.UUID) (string, string, error) {
	// Access token
	access := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":  userID,
			"exp": time.Now().Add(config.AppConfig.AccessTokenDuration).Unix(),
		},
	)
	accessToken, err := access.SignedString(config.AppConfig.JWTSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshToken := uuid.New().String()
	expiresAt := time.Now().Add(config.AppConfig.RefreshTokenDuration)

	if err := s.repo.CreateRefreshToken(refreshToken, userID, expiresAt); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) Refresh(refreshToken string) (string, string, error) {
	rt, err := s.repo.FindValidByToken(refreshToken)
	if err != nil {
		return "", "", errors.New("invalid or expired refresh token")
	}

	// Отзываем старый
	err = s.repo.Revoke(refreshToken)

	access, newRefresh, err := s.GenerateTokens(rt.UserID)
	return access, newRefresh, err
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
