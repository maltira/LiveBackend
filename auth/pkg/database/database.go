package database

import (
	"auth/config"
	"auth/internal/models"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB() {
	cfg := config.Env

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
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

func CloseDB() {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Error sql.DB: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("PostgreSQL closing error: %v", err)
	}
	db = nil
	log.Println("Database connection closed")
}
