package dto

import "github.com/google/uuid"

// * Requests

type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type VerifyOTPRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Code   string    `json:"code" binding:"required,len=6"`
	Action string    `json:"action" binding:"required"`
}

type ResetPasswordRequest struct {
	UserID      uuid.UUID `json:"user_id" binding:"required"`
	NewPassword string    `json:"new_password" binding:"required,min=6"`
}

type TempTokenRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,min=6"`
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
	Message       string `json:"message"`
	RecoveryToken string `json:"recovery_token"`
}
