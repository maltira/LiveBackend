package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	Email      string    `json:"email" gorm:"not null;uniqueIndex"`
	Password   string    `json:"-" gorm:"not null"`
	IsVerified bool      `json:"is_verified" gorm:"not null;default:false"`

	CreatedAt     time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	ToBeDeletedAt *time.Time     `json:"to_be_deleted_at" gorm:"index"`

	RefreshTokens []RefreshToken `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	OTPCodes      []OTPCode      `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
