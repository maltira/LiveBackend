package models

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"` // auth.User.ID

	Username  string     `json:"username" gorm:"size:50;uniqueIndex;not null"` // по которому можно искать
	FullName  string     `json:"full_name" gorm:"size:100;not null"`           // имя + фамилия
	Bio       string     `json:"bio" gorm:"size:500"`
	AvatarURL string     `json:"avatar_url" gorm:"size:255"`
	BirthDate *time.Time `json:"birth_date" gorm:"default:null"`
	LastSeen  time.Time  `json:"last_seen" gorm:"index;autoCreateTime;not null"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime;not null"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Settings Settings `gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE;"`
}
