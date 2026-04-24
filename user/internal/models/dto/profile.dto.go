package dto

import (
	"time"
)

type CreateProfileRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

type UpdateProfileRequest struct {
	Username  *string    `json:"username" binding:"min=4,max=16"`
	FullName  *string    `json:"full_name" binding:"min=1,max=255"`
	Bio       *string    `json:"bio" binding:"max=500"`
	AvatarURL *string    `json:"avatar_url" binding:"url"`
	BirthDate *time.Time `json:"birth_date"`
}

type ProfileStatusResponse struct {
	Online   bool       `json:"online"`
	LastSeen *time.Time `json:"last_seen,omitempty"`
}
