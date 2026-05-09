package utils

import (
	"crypto/rand"
	"crypto/sha256"
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

// HashTokenSHA256 возвращает детерминированный SHA256-хеш токена.
// Используется как ключ в Redis для O(1) lookup вместо KEYS + цикл.
// Безопасно, тк исходный токен — 32 случайных байта (256 бит энтропии).
func HashTokenSHA256(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
