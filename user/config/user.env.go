package config

import (
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	AppPort      string
	RabbitMQAddr string
	RedisAddr    string
}

var Env *Config

func InitEnv() {
	Env = &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_USER_NAME"),

		AppPort:      os.Getenv("PORT_USER"),
		RabbitMQAddr: os.Getenv("RABBITMQ_ADDR"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
	}
}
