package dto

import (
	"github.com/google/uuid"
)

// * Requests

type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type VerifyLoginOTPRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Code   string    `json:"code" binding:"required,len=6"`
}

type VerifyDeleteOTPRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Code   string    `json:"code" binding:"required,len=6"`
	Reason string    `json:"reason" binding:"max=255"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

type ChangeEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}
type ChangePasswordRequest struct {
	Password    string `json:"password" binding:"required,min=8"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// * Responses

type OTPSentResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Message string    `json:"message"`
}

type LoginResponse struct {
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	AccessToken string    `json:"access_token"`
}
