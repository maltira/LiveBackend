package database

import (
	"auth/models"
	"common/config"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() {
	cfg := config.AppConfig

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBAuthName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Автомиграция таблиц
	err = db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.OTPCode{})
	if err != nil {
		panic("failed to migrate database: " + err.Error())
	}
}

func GetDB() *gorm.DB {
	return db
}
