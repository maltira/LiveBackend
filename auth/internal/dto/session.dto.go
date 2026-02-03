package dto

import (
	"time"

	"github.com/google/uuid"
)

type SessionResponse struct {
	ID        uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Device    string    `json:"device" example:"Mobile / iOS 17 / Safari"`
	IP        string    `json:"ip" example:"85.145.12.34"`
	UserAgent string    `json:"user_agent" example:"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)..."`
	CreatedAt time.Time `json:"created_at" example:"2026-01-16T09:17:00Z"`
	ExpiresAt time.Time `json:"expires_at" example:"2026-02-16T09:17:00Z"`
}
