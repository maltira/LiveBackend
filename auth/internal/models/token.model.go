package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID     uuid.UUID `json:"-" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	UserID uuid.UUID `json:"-" gorm:"type:uuid;not null;index;"`
	Token  string    `json:"-" gorm:"not null;index;"`

	AccessJTI string `json:"-" gorm:"size:36;not null"`

	IP        string `json:"ip" gorm:"size:45"`
	UserAgent string `json:"user_agent" gorm:"size:255"`
	Device    string `json:"device" gorm:"size:100"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
}
