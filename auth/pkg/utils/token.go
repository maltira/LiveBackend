package utils

import (
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// GenerateSecureToken генерирует криптостойкий токен
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken хэширует токен для хранения
func HashToken(token string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	return string(hashed), err
}

// CompareToken сравнивает токен с хэшем
func CompareToken(token, hashed string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(token))
	return err == nil
}
