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

	DeletedAt      *time.Time `json:"deleted_at"`
	DeletionReason *string    `json:"deletion_reason"`
	DeletedBy      *string    `json:"deleted_by" gorm:"type:varchar(50);check: deleted_by IN ('user', 'system')"`

	CreatedAt         time.Time `json:"created_at" gorm:"not null;autoCreateTime"`
	PasswordUpdatedAt time.Time `json:"password_updated_at" gorm:"not null;autoCreateTime"`
	EmailUpdatedAt    time.Time `json:"email_updated_at" gorm:"not null;autoCreateTime"`

	RefreshTokens []RefreshToken `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	OTPCodes      []OTPCode      `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
