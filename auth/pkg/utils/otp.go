package utils

import (
	"auth/config"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
)

func GenerateOTP() string {
	const digits = "0123456789"
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		b[i] = digits[n.Int64()]
	}
	return string(b)
}

// HashOTP хеширует OTP-код через HMAC-SHA256 с серверным секретом.
// Детерминистичный хеш позволяет искать по нему в SQL, а HMAC с секретом
// предотвращает brute-force 6-значного кода при утечке БД.
func HashOTP(code string) string {
	mac := hmac.New(sha256.New, config.Env.JWTSecret)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

