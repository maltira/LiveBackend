package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PortAuth string
	PortUser string
	PortChat string
	AppPort  string
}

var Env *Config

func InitEnv() {
	if err := godotenv.Load("../.env"); err != nil {
		panic(".env file not found")
	}

	Env = &Config{
		PortAuth: os.Getenv("PORT_AUTH"),
		PortUser: os.Getenv("PORT_USER"),
		PortChat: os.Getenv("PORT_CHAT"),
		AppPort:  os.Getenv("PORT_USER"),
	}
}
