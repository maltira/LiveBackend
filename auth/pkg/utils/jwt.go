package utils

import (
	"auth/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func ParseToken(tokenString string) (*jwt.Token, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return config.Env.JWTSecret, nil
	})
}

func GenerateRefreshTokenString() (string, time.Time) {
	refreshToken := uuid.NewString()
	expiresAt := time.Now().Add(config.Env.RefreshTokenDuration)
	return refreshToken, expiresAt
}

func GenerateAccessToken(userID uuid.UUID) (string, error) {
	access := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":  userID,
			"exp": time.Now().Add(config.Env.AccessTokenDuration).Unix(),
		},
	)
	return access.SignedString(config.Env.JWTSecret)
}
