package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret []byte

	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBSSLMode  string

	DBAuthName string
	DBUserName string
	DBChatName string

	PortAuth    string
	PortUser    string
	PortChat    string
	PortGateway string
}

var AppConfig *Config

func Load() {
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env file not found")
	}

	AppConfig = &Config{
		JWTSecret: []byte(os.Getenv("JWT_SECRET")),

		AccessTokenDuration:  getDuration("JWT_ACCESS_DURATION", 15*time.Minute),
		RefreshTokenDuration: getDuration("JWT_REFRESH_DURATION", 30*24*time.Hour),

		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),

		DBAuthName: os.Getenv("DB_AUTH_NAME"),
		DBUserName: os.Getenv("DB_USER_NAME"),
		DBChatName: os.Getenv("DB_CHAT_NAME"),

		PortAuth:    os.Getenv("PORT_AUTH"),
		PortUser:    os.Getenv("PORT_USER"),
		PortChat:    os.Getenv("PORT_CHAT"),
		PortGateway: os.Getenv("PORT_GATEWAY"),
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
