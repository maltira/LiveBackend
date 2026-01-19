package database

import (
	"common/config"
	"fmt"
	"log"
	"user/models"

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
		cfg.DBUserName,
		cfg.DBPort,
		cfg.DBSSLMode,
	)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect to database: " + err.Error())
	}

	// Автомиграция таблиц
	err = db.AutoMigrate(&models.Profile{}, &models.Block{})
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
		log.Printf("Ошибка получения sql.DB: %v", err)
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("Ошибка закрытия PostgreSQL: %v", err)
	}
	db = nil
	fmt.Println("database connection closed")
}
