package config

import (
	"os"
)

type Config struct {
	JWTSecret []byte
	RedisAddr string
	PortAuth  string
	PortUser  string
	PortChat  string
	AppPort   string
}

var Env *Config

func InitEnv() {
	Env = &Config{
		JWTSecret: []byte(os.Getenv("JWT_SECRET")),
		RedisAddr: os.Getenv("REDIS_ADDR"),
		PortAuth:  os.Getenv("PORT_AUTH"),
		PortUser:  os.Getenv("PORT_USER"),
		PortChat:  os.Getenv("PORT_CHAT"),
		AppPort:   os.Getenv("PORT_GATEWAY"),
	}
}
