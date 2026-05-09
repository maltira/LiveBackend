package utils

import (
	"auth/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateRefreshTokenString() (string, time.Time) {
	refreshToken := uuid.NewString()
	expiresAt := time.Now().Add(config.Env.RefreshTokenDuration)
	return refreshToken, expiresAt
}

func GenerateAccessToken(userID uuid.UUID) (string, string, error) {
	jti := uuid.NewString()
	access := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.MapClaims{
			"id":  userID,
			"jti": jti,
			"exp": time.Now().Add(config.Env.AccessTokenDuration).Unix(),
		},
	)
	token, err := access.SignedString(config.Env.JWTSecret)
	return token, jti, err
}
