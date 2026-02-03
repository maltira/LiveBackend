package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret []byte

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBSSLMode  string
	DBName     string

	AppPort      string
	RabbitMQAddr string
	RedisAddr    string
}

var Env *Config

func InitEnv() {
	if err := godotenv.Load("../.env"); err != nil {
		panic(".env file not found")
	}

	Env = &Config{
		JWTSecret: []byte(os.Getenv("JWT_SECRET")),

		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBSSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:     os.Getenv("DB_USER_NAME"),

		AppPort:      os.Getenv("PORT_USER"),
		RabbitMQAddr: os.Getenv("RABBITMQ_ADDR"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
	}
}
