package dto

// * Requests

type AuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type VerifyOTPRequest struct {
	Action    string `json:"action" binding:"required"`
	TempToken string `json:"temp_token" binding:"required"`
	Code      string `json:"code" binding:"required,len=6"`
}

// * Responses

type AuthResponse struct {
	Action    string `json:"action"`
	TempToken string `json:"temp_token"`
}
