package models

import (
	"time"

	"github.com/google/uuid"
)

type OTPCode struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	UserID uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	Code   string    `json:"-" gorm:"not null;size:64"`
	IsUsed bool      `json:"is_used" gorm:"default:false"`

	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}
