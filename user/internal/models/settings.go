package models

import "github.com/google/uuid"

type Settings struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`
	ProfileID uuid.UUID `json:"profile_id" gorm:"type:uuid;not null;index"`

	ShowOnlineStatus bool   `json:"show_online_status" gorm:"default:true;not null"`
	ShowBirthDate    string `json:"show_birth_date" gorm:"default:'all';not null;check:show_birth_date IN ('all', 'nobody')"`

	DarkMode bool   `json:"dark_mode" gorm:"default:false;not null"`
	Language string `json:"language" gorm:"default:'ru-RU';not null;check:language IN ('ru-RU', 'en-US')"`
}
