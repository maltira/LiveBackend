package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID  `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	ChatID         uuid.UUID  `json:"chat_id" gorm:"type:uuid;index;not null"`
	UserID         *uuid.UUID `json:"user_id" gorm:"type:uuid;index;not null"` // null = система
	Content        string     `json:"content" gorm:"type:text;not null"`
	Type           string     `json:"type" gorm:"default:'text';check: type IN ('text', 'image', 'video', 'file', 'system');not null"`
	ReplyToMessage *uuid.UUID `json:"reply_to_message"`

	CreatedAt time.Time  `json:"created_at" gorm:"index:idx_chat_messages;sort:DESC;not null"`
	EditedAt  *time.Time `json:"edited_at"`

	Chat Chat `gorm:"foreignKey:ChatID;constraint:OnDelete:CASCADE;"`
}
