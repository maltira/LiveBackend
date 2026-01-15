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

	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at"`

	RefreshTokens []RefreshToken `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	OTPCodes      []OTPCode      `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"-" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	UserID    uuid.UUID `json:"-" gorm:"type:uuid;not null;index;"`
	Token     string    `json:"-" gorm:"not null"`
	ExpiresAt time.Time `json:"-" gorm:"not null"`
	IP        string    `gorm:"size:45"`
	UserAgent string    `gorm:"size:255"`
	Device    string    `gorm:"size:100"`
}

type OTPCode struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	UserID uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Code   string    `json:"code" gorm:"not null;size:6"`
	IsUsed bool      `json:"is_used" gorm:"default:false"`

	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
