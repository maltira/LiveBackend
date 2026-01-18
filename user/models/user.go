package models

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"` // auth.User.ID

	Username  string `json:"username" gorm:"size:50;uniqueIndex"` // по которому можно искать
	FullName  string `json:"full_name" gorm:"size:100"`           // имя + фамилия
	Bio       string `json:"bio" gorm:"size:500"`
	AvatarURL string `json:"avatar_url" gorm:"size:255"`
	BirthDate *time.Time

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}
