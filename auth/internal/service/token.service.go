package service

import (
	"auth/config"
	"auth/internal/models"
	"auth/internal/repository"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenService interface {
	GenerateTokens(userID uuid.UUID, ip, userAgent, device string) (string, string, error)
	Refresh(refreshToken string) (string, string, error)
	RevokeRefreshToken(refreshToken string) error
	RevokeAllRefreshTokens(userID uuid.UUID, excludeToken *string) error
	ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error)
}
type tokenService struct {
	repo repository.TokenRepository
}

func NewTokenService(repo repository.TokenRepository) TokenService {
	return &tokenService{repo: repo}
}

func (s *tokenService) GenerateTokens(userID uuid.UUID, ip, userAgent, device string) (string, string, error) {
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

func (s *tokenService) Refresh(refreshToken string) (string, string, error) {
	rt, err := s.repo.FindValidByToken(refreshToken)
	if err != nil {
		return "", "", err
	}

	// Отзываем старый
	_ = s.repo.Revoke(refreshToken)

	access, newRefresh, err := s.GenerateTokens(rt.UserID, rt.IP, rt.UserAgent, rt.Device)
	return access, newRefresh, err
}

func (s *tokenService) RevokeRefreshToken(refreshToken string) error {
	return s.repo.Revoke(refreshToken)
}

func (s *tokenService) RevokeAllRefreshTokens(userID uuid.UUID, excludeToken *string) error {
	return s.repo.RevokeAll(userID, excludeToken)
}

func (s *tokenService) ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error) {
	return s.repo.ListActiveSessions(userID)
}
