package dto

// * Requests

type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type VerifyOTPRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
	Code      string `json:"code" binding:"required,len=6"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
	ResetToken  string `json:"reset_token" binding:"required"`
}

type TempTokenRequest struct {
	TempToken string `json:"temp_token" binding:"required"`
}

type DeleteAccountRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// * Responses

type AuthResponse struct {
	TempToken string `json:"temp_token"`
}

type RecoveryResponse struct {
	Message       string `json:"message"`
	RecoveryToken string `json:"recovery_token"`
}
