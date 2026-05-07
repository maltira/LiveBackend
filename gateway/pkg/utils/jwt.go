package utils

import (
	"errors"
	"gateway/config"

	"github.com/golang-jwt/jwt/v5"
)

func ParseToken(tokenString string) (*jwt.Token, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return config.Env.JWTSecret, nil
	})
}
