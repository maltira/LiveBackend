package dto

import (
	"github.com/google/uuid"
)

// * Requests

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,strong_password"`
}
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
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
type VerifyChangeEmailRequest struct {
	UserID   uuid.UUID `json:"user_id" binding:"required"`
	Code     string    `json:"code" binding:"required,len=6"`
	NewEmail string    `json:"new_email" binding:"required,email"`
}

type VerifyChangePassRequest struct {
	UserID      uuid.UUID `json:"user_id" binding:"required"`
	Code        string    `json:"code" binding:"required,len=6"`
	Password    string    `json:"password" binding:"required"`
	NewPassword string    `json:"new_password" binding:"required,nefield=Password,strong_password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required"`
	Password        string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password,strong_password"`
}

type SendOTPRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required,uuid"`
	Email  string    `json:"email" binding:"required,email"`
}

type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" binding:"required,email"`
}

type ChangePasswordRequest struct {
	Password    string `json:"password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,strong_password,nefield=Password"`
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
