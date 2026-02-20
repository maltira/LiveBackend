package dto

import (
	"time"

	"github.com/google/uuid"
)

// * Requests

type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type VerifyOTPRequest struct {
	UserID   uuid.UUID `json:"user_id" binding:"required"`
	Email    *string   `json:"email"`
	Password *string   `json:"password"`
	Code     string    `json:"code" binding:"required,len=6"`
	Action   string    `json:"action" binding:"required"`
}

type ResetPasswordRequest struct {
	UserID      uuid.UUID `json:"user_id" binding:"required"`
	NewPassword string    `json:"new_password" binding:"required,min=8"`
}

type TempTokenRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,min=8"`
}

type ChangeEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}
type ChangePasswordRequest struct {
	Password    string `json:"password" binding:"required,min=8"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// * Responses

type AuthResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Message string    `json:"message"`
}

type OTPSentResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Message string    `json:"message"`
}

type TempTokenResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	TempToken string    `json:"temp_token"`
}

type RecoveryResponse struct {
	ToBeDeletedAt time.Time `json:"to_be_deleted_at"`
	RecoveryToken string    `json:"recovery_token"`
}
