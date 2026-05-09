package service

import (
	"auth/config"
	"auth/internal/models"
	"auth/internal/repository"
	authRedis "auth/pkg/redis"
	"auth/pkg/utils"
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

type TokenService interface {
	GenerateTokens(userID uuid.UUID, ip, userAgent, device string) (string, string, error)
	Refresh(refreshToken string) (string, string, error)
	RevokeRefreshToken(refreshToken string) error
	RevokeRefreshTokenById(userID, tokenID uuid.UUID) error
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

	// Blacklist старый access jti
	if rt.AccessJTI != "" {
		_ = BlacklistJTI(rt.AccessJTI)
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

	accessToken, jti, err := utils.GenerateAccessToken(userID)
	if err != nil {
		return "", "", err
	}

	if err = s.repo.SaveRefreshToken(refreshToken, userID, expiresAt, ip, userAgent, device, jti); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *tokenService) RevokeRefreshToken(refreshToken string) error {
	// Получаем запись, чтобы достать jti для blacklist
	rt, err := s.repo.FindRefreshToken(refreshToken)
	if err == nil && rt.AccessJTI != "" {
		_ = BlacklistJTI(rt.AccessJTI)
	}
	return s.repo.Revoke(refreshToken)
}
func (s *tokenService) RevokeRefreshTokenById(userID, tokenID uuid.UUID) error {
	// Получаем запись, чтобы достать jti для blacklist
	rt, err := s.repo.FindRefreshTokenById(tokenID)
	if rt.UserID != userID {
		return errors.New("invalid action")
	}
	if err == nil && rt.AccessJTI != "" {
		_ = BlacklistJTI(rt.AccessJTI)
	}
	return s.repo.Revoke(rt.Token)
}

func (s *tokenService) RevokeAllRefreshTokens(userID uuid.UUID, excludeToken *string) error {
	// Собираем все jti перед удалением
	jtis, err := s.repo.GetActiveJTIs(userID, excludeToken)
	if err != nil {
		log.Printf("Warning: failed to get JTIs for blacklist: %v", err)
	}
	for _, jti := range jtis {
		_ = BlacklistJTI(jti)
	}
	return s.repo.RevokeAll(userID, excludeToken)
}

func (s *tokenService) ListActiveSessions(userID uuid.UUID) ([]models.RefreshToken, error) {
	return s.repo.ListActiveSessions(userID)
}

// BlacklistJTI помещает jti access-токена в Redis blacklist с TTL = AccessTokenDuration
func BlacklistJTI(jti string) error {
	ctx := context.Background()
	key := "blacklist:jti:" + jti
	// TTL = AccessTokenDuration, после истечения access-токен и так станет невалидным
	return authRedis.AuthRedis.Set(ctx, key, "1", config.Env.AccessTokenDuration).Err()
}
