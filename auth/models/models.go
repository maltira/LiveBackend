package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	Email      string    `json:"email" gorm:"not null;uniqueIndex"`
	Password   string    `json:"-" gorm:"not null"`
	IsVerified bool      `json:"is_verified" gorm:"not null;default:false"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"-" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	UserID    uuid.UUID `json:"-" gorm:"type:uuid;not null;index;"`
	Token     string    `json:"-" gorm:"not null"`
	ExpiresAt time.Time `json:"-" gorm:"not null"`
	Revoked   bool      `json:"-" gorm:"default:false"`

	// связи
	User User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type OTPCode struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	UserID uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Code   string    `json:"code" gorm:"not null;size:6"`
	IsUsed bool      `json:"is_used" gorm:"default:false"`

	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	// связи
	User User `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
