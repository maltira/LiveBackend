package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type AuthConfig struct {
	JWTSecret            []byte
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBSSLMode  string
	DBName     string

	AppPort      string
	RedisAddr    string
	RabbitMQAddr string
}

var Env *AuthConfig

func InitEnv() {
	if err := godotenv.Load("../.env"); err != nil {
		panic(".env file not found")
	}

	Env = &AuthConfig{
		JWTSecret: []byte(os.Getenv("JWT_SECRET")),

		AccessTokenDuration:  getDuration("JWT_ACCESS_DURATION", 15*time.Minute),
		RefreshTokenDuration: getDuration("JWT_REFRESH_DURATION", 30*24*time.Hour),

		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:     os.Getenv("DB_AUTH_NAME"),

		AppPort:      os.Getenv("PORT_AUTH"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
		RabbitMQAddr: os.Getenv("RABBITMQ_ADDR"),
	}
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if dur, err := time.ParseDuration(value); err == nil {
			return dur
		}
		log.Printf("Invalid duration for %s, using fallback", key)
	}
	return fallback
}
