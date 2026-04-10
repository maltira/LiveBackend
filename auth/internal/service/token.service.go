package service

import (
	"auth/internal/models"
	"auth/internal/repository"
	"auth/pkg/utils"
	"errors"
	"time"

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

func (s *tokenService) Refresh(refreshToken string) (string, string, error) {
	rt, err := s.repo.FindRefreshToken(refreshToken)
	if err != nil || time.Now().After(rt.ExpiresAt) {
		return "", "", errors.New("invalid refresh token")
	}
	// Отзываем старый токен
	_ = s.repo.Revoke(refreshToken)

	// Создаём новые
	access, newRefresh, err := s.GenerateTokens(rt.UserID, rt.IP, rt.UserAgent, rt.Device)
	if err != nil {
		return "", "", err
	}

	return access, newRefresh, nil
}

func (s *tokenService) GenerateTokens(userID uuid.UUID, ip, userAgent, device string) (string, string, error) {
	refreshToken, expiresAt := utils.GenerateRefreshTokenString()

	accessToken, err := utils.GenerateAccessToken(userID)
	if err != nil {
		return "", "", err
	}

	if err = s.repo.SaveRefreshToken(refreshToken, userID, expiresAt, ip, userAgent, device); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
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
