package dto

import (
	"time"
)

type UpdateProfileRequest struct {
	Username  string     `json:"username"`
	FullName  string     `json:"full_name"`
	Bio       string     `json:"bio"`
	AvatarURL string     `json:"avatar_url"`
	BirthDate *time.Time `json:"birth_date"`
}
