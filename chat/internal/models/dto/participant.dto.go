package dto

import (
	"time"

	"github.com/google/uuid"
)

type MuteRequest struct {
	MutedID   uuid.UUID `json:"muted_id"`
	UntilDate time.Time `json:"until_date"`
}
