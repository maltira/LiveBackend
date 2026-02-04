package models

import (
	"time"

	"github.com/google/uuid"
)

type Participant struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	ChatID uuid.UUID `json:"chat_id" gorm:"type:uuid;uniqueIndex:idx_participant;not null"`
	UserID uuid.UUID `json:"user_id" gorm:"type:uuid;uniqueIndex:idx_participant;not null"`
	Role   string    `json:"role" gorm:"type:varchar(20);default:'member';check:role IN ('member', 'admin', 'owner')"`

	JoinedAt   time.Time  `json:"joined_at" gorm:"autoCreateTime;not null"`
	MutedUntil *time.Time `json:"muted_until"` // для групп

	Chat Chat `gorm:"foreignKey:ChatID;constraint:OnDelete:CASCADE;"`
}
