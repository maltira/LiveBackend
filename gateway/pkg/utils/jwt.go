package utils

import (
	"errors"
	"fmt"
	"gateway/config"

	"github.com/golang-jwt/jwt/v5"
)

func ParseToken(tokenString string) (*jwt.Token, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}
	return jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return config.Env.JWTSecret, nil
	})
}
