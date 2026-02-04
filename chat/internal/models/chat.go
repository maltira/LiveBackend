package models

import (
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	Type      string    `json:"type" gorm:"type:varchar(20);default:'private';check: type IN ('private', 'group'); not null"`
	Name      *string   `json:"name" gorm:"size:100;default:null"`       // для групп
	AvatarURL *string   `json:"avatar_url" gorm:"size:255;default:null"` // для групп
	CanJoin   *bool     `json:"can_join"`                                // для групп
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:uuid;index;not null"`

	LastMessageAt time.Time `json:"last_message_at" gorm:"autoCreateTime;index;not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime;not null"`
}
