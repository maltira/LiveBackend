package dto

import "github.com/google/uuid"

type ChatCreateRequest struct {
	Name      string      `json:"name" binding:"required,min=1,max=75"`
	AvatarURL *string     `json:"avatar_url" binding:"max=255"`
	CanJoin   bool        `json:"can_join"`
	Members   []uuid.UUID `json:"members"`
}
