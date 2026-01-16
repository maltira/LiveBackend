package utils

import (
	"auth/models"
	"time"

	"gorm.io/gorm"
)

func DeleteExpiredAccounts(db *gorm.DB) {
	var users []models.User
	db.Where("to_be_deleted_at <= ?", time.Now()).
		Find(&users)

	for _, u := range users {
		db.Delete(&u)
	}
}
