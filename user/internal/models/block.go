package models

import (
	"time"

	"github.com/google/uuid"
)

type Block struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;default:gen_random_uuid();primaryKey; not null"`

	ProfileID        uuid.UUID `json:"profile_id" gorm:"type:uuid;not null;uniqueIndex:idx_block_pair"`
	BlockedProfileID uuid.UUID `json:"blocked_profile_id" gorm:"type:uuid;not null;uniqueIndex:idx_block_pair"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	Profile        Profile `json:"-" gorm:"foreignKey:ProfileID;constraint:OnDelete:CASCADE;"`
	BlockedProfile Profile `gorm:"foreignKey:BlockedProfileID;constraint:OnDelete:CASCADE;"`
}
